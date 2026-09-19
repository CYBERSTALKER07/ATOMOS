let meta = #{
    name: "graph-retrieve",
    description: "Graph neighborhood + live-code verify + durable memory write (no side vector DB)",
    when_to_use: "RAG, graph retrieval, remember this, where does X live, agent memory loop",
    phases: [
        #{ title: "Retrieve", detail: "graph walk + code grep" },
        #{ title: "Verify", detail: "adversarial live-path check" },
        #{ title: "Persist", detail: "workspace MEMORY only if verified" },
    ],
};

let retrieve_schema = #{
    "type": "object",
    "required": ["paths", "notes"],
    "properties": #{
        "paths": #{
            "type": "array",
            "maxItems": 16,
            "items": #{ "type": "string" },
        },
        "notes": #{ "type": "string" },
    },
};

let verify_schema = #{
    "type": "object",
    "required": ["real", "fact", "evidence"],
    "properties": #{
        "real": #{ "type": "boolean" },
        "fact": #{ "type": "string" },
        "evidence": #{ "type": "string" },
    },
};

let topic = if args == () { () } else { args.topic };
if topic == () { pause("verification", "Pass args.topic — the capability, route, or package to retrieve."); }

phase("Retrieve");
let q = json_encode(topic);
let retriever_prompt = "";
retriever_prompt += "You are a retrieval agent for pegasusX. Topic (JSON): ";
retriever_prompt += q;
retriever_prompt += " Use tools. Run: python3 .agents/skills/graph-retrieval-memory/scripts/graph_retrieve.py --query ";
retriever_prompt += q;
retriever_prompt += " --hops 2";
retriever_prompt += " Then grep/read the returned paths under pegasusX/ (not pegasus/). ";
retriever_prompt += "Also try to read ~/.grok/memory/MEMORY.md if it exists. ";
retriever_prompt += "Return at most 16 repo-relative file paths you actually opened, plus short notes. ";
retriever_prompt += "Empty paths only if the graph walker ran and no files existed.";

let retrieved = agent(retriever_prompt, #{
    label: "retrieve",
    capability_mode: "read-only",
    output_schema: retrieve_schema,
});

if retrieved == () || !retrieved.success {
    complete(#{ summary: "Retrieve agent failed", verified: [] });
}

phase("Verify");
let vjobs = [];
let paths = retrieved.output.paths;
if paths == () { paths = []; }
let i = 0;
for p in paths {
    if i >= 8 { break; }
    let vp = "";
    vp += "Adversarially verify what is TRUE in current pegasusX code about topic ";
    vp += q;
    vp += " by reading ";
    vp += json_encode(p);
    vp += ". Set real=true only with file:line you opened this turn. ";
    vp += "fact must be one durable sentence. evidence must include path:line. ";
    vp += "If the file is missing or docs-only, real=false.";
    vjobs.push(#{
        prompt: vp,
        label: "verify:" + i.to_string(),
        capability_mode: "read-only",
        output_schema: verify_schema,
    });
    i += 1;
}
if vjobs.len() == 0 {
    complete(#{ summary: "No paths to verify", notes: retrieved.output.notes, verified: [] });
}

let verdicts = parallel(vjobs);
let verified = [];
for v in verdicts {
    if v != () && v.success && v.output.real == true
        && v.output.fact != () && v.output.fact != ""
        && v.output.evidence != () && v.output.evidence != "" {
        verified.push(#{ fact: v.output.fact, evidence: v.output.evidence });
    }
}

phase("Persist");
if verified.len() == 0 {
    complete(#{
        summary: "Nothing verified; no memory write",
        notes: retrieved.output.notes,
        verified: [],
    });
}

let persist_prompt = "";
persist_prompt += "Append ONLY these verified facts to workspace Grok memory if [memory] is enabled. ";
persist_prompt += "Read ~/.grok/memory/ first. If a workspace MEMORY.md exists (ATOMOS/VOID/pegasus slug), ";
persist_prompt += "append a ## Verified section with each fact + evidence. Do not invent extra facts. ";
persist_prompt += "If memory dir is missing, say so and do not create a fake store. Facts JSON: ";
persist_prompt += json_encode(verified);

let persisted = agent(persist_prompt, #{
    label: "persist",
    capability_mode: "read-write",
});

let persist_ok = persisted != () && persisted.success;
complete(#{
    summary: verified.len().to_string() + " verified facts",
    verified: verified,
    persist_ok: persist_ok,
});
