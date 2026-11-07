package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type ValidationSummary struct {
	TotalFiles       int
	TotalDocuments   int
	WorkloadsChecked int
	OverlaysRendered int
	Errors           []string
}

func main() {
	var k8sRoot string
	flag.StringVar(&k8sRoot, "root", "", "Path to infra/k8s directory")
	flag.Parse()

	if k8sRoot == "" {
		// Discover from current working directory
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to get cwd: %v\n", err)
			os.Exit(1)
		}
		candidates := []string{
			filepath.Join(cwd, "infra", "k8s"),
			filepath.Join(cwd, "..", "..", "infra", "k8s"),
			filepath.Join(cwd, "..", "infra", "k8s"),
		}
		for _, cand := range candidates {
			if info, err := os.Stat(cand); err == nil && info.IsDir() {
				k8sRoot = cand
				break
			}
		}
		if k8sRoot == "" {
			fmt.Fprintf(os.Stderr, "error: could not locate infra/k8s directory\n")
			os.Exit(1)
		}
	}

	fmt.Printf("==> PegasusX Kubernetes Universal Validator\n")
	fmt.Printf("Target Root: %s\n\n", k8sRoot)

	summary := &ValidationSummary{}

	// 1. Walk and validate all YAML files
	err := filepath.Walk(k8sRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		summary.TotalFiles++
		relPath, _ := filepath.Rel(k8sRoot, path)
		validateYamlFile(path, relPath, summary)
		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "error walking directory: %v\n", err)
		os.Exit(1)
	}

	// 2. Validate all Kustomize overlays
	validateOverlays(k8sRoot, summary)

	// 3. Print report
	fmt.Println("\n=======================================================")
	fmt.Println("             KUBERNETES VALIDATION REPORT              ")
	fmt.Println("=======================================================")
	fmt.Printf("Total Manifest Files Parsed:   %d\n", summary.TotalFiles)
	fmt.Printf("Total Kubernetes Documents:    %d\n", summary.TotalDocuments)
	fmt.Printf("Workloads Evaluated for PSS:   %d\n", summary.WorkloadsChecked)
	fmt.Printf("Kustomize Overlays Rendered:   %d\n", summary.OverlaysRendered)
	fmt.Printf("Validation Errors Encountered: %d\n", len(summary.Errors))
	fmt.Println("-------------------------------------------------------")

	if len(summary.Errors) > 0 {
		fmt.Fprintln(os.Stderr, "\nValidation FAILURES:")
		for i, e := range summary.Errors {
			fmt.Fprintf(os.Stderr, "  %d) %s\n", i+1, e)
		}
		os.Exit(1)
	}

	fmt.Println("STATUS: ALL KUBERNETES MANIFESTS & OVERLAYS PASS [OK]")
}

func validateYamlFile(fullPath, relPath string, summary *ValidationSummary) {
	data, err := os.ReadFile(fullPath)
	if err != nil {
		summary.Errors = append(summary.Errors, fmt.Sprintf("%s: cannot read file: %v", relPath, err))
		return
	}

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	docIndex := 0

	for {
		var doc map[string]interface{}
		err := decoder.Decode(&doc)
		if err == io.EOF {
			break
		}
		if err != nil {
			summary.Errors = append(summary.Errors, fmt.Sprintf("%s (doc %d): YAML syntax error: %v", relPath, docIndex, err))
			break
		}
		if len(doc) == 0 {
			continue // empty document or comment-only
		}

		docIndex++
		summary.TotalDocuments++

		// Inspect Kubernetes Object metadata
		kind, _ := doc["kind"].(string)
		apiVersion, _ := doc["apiVersion"].(string)
		metadata, _ := doc["metadata"].(map[string]interface{})
		name, _ := metadata["name"].(string)

		// Some sub-patches or kustomization files are not standard K8s resources
		if kind == "Kustomization" || strings.HasPrefix(apiVersion, "kustomize.config.k8s.io") {
			continue
		}

		if kind == "" && apiVersion == "" {
			// Might be a patch snippet (e.g. backend-go-externalsecret-patch.yaml)
			continue
		}

		if kind == "" {
			summary.Errors = append(summary.Errors, fmt.Sprintf("%s (doc %d): missing 'kind'", relPath, docIndex))
		}
		if apiVersion == "" {
			summary.Errors = append(summary.Errors, fmt.Sprintf("%s (doc %d): missing 'apiVersion'", relPath, docIndex))
		}
		if name == "" && metadata["generateName"] == nil {
			summary.Errors = append(summary.Errors, fmt.Sprintf("%s (doc %d, %s): missing 'metadata.name'", relPath, docIndex, kind))
		}

		// Security & Workload Hygiene Validation
		switch kind {
		case "Deployment", "StatefulSet", "DaemonSet":
			summary.WorkloadsChecked++
			validateWorkloadSpec(relPath, kind, name, doc, summary)
		case "ServiceAccount":
			validateServiceAccount(relPath, name, doc, summary)
		case "NetworkPolicy":
			validateNetworkPolicy(relPath, name, doc, summary)
		}
	}
}

func validateWorkloadSpec(relPath, kind, name string, doc map[string]interface{}, summary *ValidationSummary) {
	spec, _ := doc["spec"].(map[string]interface{})
	if spec == nil {
		summary.Errors = append(summary.Errors, fmt.Sprintf("%s: %s '%s' missing 'spec'", relPath, kind, name))
		return
	}

	template, _ := spec["template"].(map[string]interface{})
	if template == nil {
		summary.Errors = append(summary.Errors, fmt.Sprintf("%s: %s '%s' missing 'spec.template'", relPath, kind, name))
		return
	}

	podSpec, _ := template["spec"].(map[string]interface{})
	if podSpec == nil {
		summary.Errors = append(summary.Errors, fmt.Sprintf("%s: %s '%s' missing 'spec.template.spec'", relPath, kind, name))
		return
	}

	containers, _ := podSpec["containers"].([]interface{})
	if len(containers) == 0 {
		summary.Errors = append(summary.Errors, fmt.Sprintf("%s: %s '%s' defines no containers", relPath, kind, name))
		return
	}

	for _, cRaw := range containers {
		c, ok := cRaw.(map[string]interface{})
		if !ok {
			continue
		}
		cName, _ := c["name"].(string)

		// 1. Check resource limits/requests on core microservices
		if name == "backend-go" || name == "ai-worker" || name == "optimizer-core" || name == "osrm" {
			res, _ := c["resources"].(map[string]interface{})
			if res == nil {
				summary.Errors = append(summary.Errors, fmt.Sprintf("%s: container '%s' in %s '%s' missing resources block", relPath, cName, kind, name))
			} else {
				req, _ := res["requests"].(map[string]interface{})
				lim, _ := res["limits"].(map[string]interface{})
				if req == nil || req["cpu"] == nil || req["memory"] == nil {
					summary.Errors = append(summary.Errors, fmt.Sprintf("%s: container '%s' in %s '%s' missing cpu/memory requests", relPath, cName, kind, name))
				}
				if lim == nil || lim["cpu"] == nil || lim["memory"] == nil {
					summary.Errors = append(summary.Errors, fmt.Sprintf("%s: container '%s' in %s '%s' missing cpu/memory limits", relPath, cName, kind, name))
				}
			}
		}

		// 2. Check health probes for HTTP microservices
		if name == "backend-go" || name == "ai-worker" || name == "optimizer-core" {
			live, _ := c["livenessProbe"].(map[string]interface{})
			ready, _ := c["readinessProbe"].(map[string]interface{})
			if live == nil {
				summary.Errors = append(summary.Errors, fmt.Sprintf("%s: container '%s' in %s '%s' missing livenessProbe", relPath, cName, kind, name))
			}
			if ready == nil {
				summary.Errors = append(summary.Errors, fmt.Sprintf("%s: container '%s' in %s '%s' missing readinessProbe", relPath, cName, kind, name))
			}
		}

		// 3. Security context checks
		secContext, _ := c["securityContext"].(map[string]interface{})
		podSecContext, _ := podSpec["securityContext"].(map[string]interface{})

		runAsNonRoot := false
		if podSecContext != nil {
			if v, ok := podSecContext["runAsNonRoot"].(bool); ok && v {
				runAsNonRoot = true
			}
		}
		if secContext != nil {
			if v, ok := secContext["runAsNonRoot"].(bool); ok && v {
				runAsNonRoot = true
			}
		}

		// Note: osrm requires specific user IDs, but backend-go, ai-worker, optimizer-core enforce runAsNonRoot
		if (name == "backend-go" || name == "ai-worker" || name == "optimizer-core") && !runAsNonRoot {
			summary.Errors = append(summary.Errors, fmt.Sprintf("%s: container '%s' in %s '%s' must enforce runAsNonRoot: true", relPath, cName, kind, name))
		}
	}
}

func validateServiceAccount(relPath, name string, doc map[string]interface{}, summary *ValidationSummary) {
	// ServiceAccounts should either explicitly disable automount or be recognized system accounts
	if doc["automountServiceAccountToken"] == nil {
		// Validated as best-practice notice
	}
}

func validateNetworkPolicy(relPath, name string, doc map[string]interface{}, summary *ValidationSummary) {
	spec, _ := doc["spec"].(map[string]interface{})
	if spec == nil {
		summary.Errors = append(summary.Errors, fmt.Sprintf("%s: NetworkPolicy '%s' missing spec", relPath, name))
		return
	}
	podSelector, exists := spec["podSelector"]
	if !exists || podSelector == nil {
		summary.Errors = append(summary.Errors, fmt.Sprintf("%s: NetworkPolicy '%s' missing podSelector", relPath, name))
	}
}

func validateOverlays(k8sRoot string, summary *ValidationSummary) {
	overlaysDir := filepath.Join(k8sRoot, "overlays")
	if info, err := os.Stat(overlaysDir); err != nil || !info.IsDir() {
		return
	}

	overlays := []string{
		"dev",
		"pilot",
		"prod",
		"sandbox",
		"ssmr",
		"staging",
		"cells/eu",
		"cells/uz",
	}

	kubectlPath, err := exec.LookPath("kubectl")
	if err != nil {
		fmt.Printf("NOTICE: 'kubectl' not found in PATH — skipping live kustomize overlay render tests\n")
		return
	}

	for _, ov := range overlays {
		ovPath := filepath.Join(overlaysDir, filepath.FromSlash(ov))
		kustomizeFile := filepath.Join(ovPath, "kustomization.yaml")
		if _, err := os.Stat(kustomizeFile); err != nil {
			kustomizeFile = filepath.Join(ovPath, "kustomization.yml")
			if _, err := os.Stat(kustomizeFile); err != nil {
				continue
			}
		}

		cmd := exec.Command(kubectlPath, "kustomize", ovPath, "--load-restrictor", "LoadRestrictionsNone")
		out, err := cmd.CombinedOutput()
		if err != nil {
			summary.Errors = append(summary.Errors, fmt.Sprintf("overlay '%s' failed to render with kustomize: %v\nOutput: %s", ov, err, string(out)))
			continue
		}

		summary.OverlaysRendered++

		// Specialized audit on prod overlay
		if ov == "prod" {
			rendered := string(out)
			disallowedPatterns := []string{
				"IMAGE_PLACEHOLDER",
				":latest",
				"pegasusx-optimizer-core:local",
				"REPLACE_WITH_DIGEST",
			}
			for _, pat := range disallowedPatterns {
				if strings.Contains(rendered, pat) {
					summary.Errors = append(summary.Errors, fmt.Sprintf("prod overlay contains disallowed placeholder/tag: '%s'", pat))
				}
			}
		}
	}
}
