# GCP Migration & Wire-Ready Checklist

*Status: Waiting on Support for quota before proceeding to full implementation.*

- [x] **0** Local make wire-ready (re-check)
- [x] **1** GCP identity + project
- [x] **2** Terraform: Spanner, Redis, GKE, VPC, AR, GSM, budget
- [ ] **3** **Spanner full schema + fiscal migrations 🔄 (you are here, ~16 tables)**
- [ ] **4** Redis AUTH → GSM + in-VPC PING (instance exists)
- [ ] **5** Confluent Kafka + topics + wire GSM
- [ ] **6** Real secrets + External Secrets
- [ ] **7** kubectl credentials
- [ ] **8** Build/push images to AR
- [ ] **9** Deploy API + worker + ai-worker
- [ ] **10** Cloud smoke: order → FISCALIZING → COMPLETED (FAKE)
- [ ] **11** Ingress + DNS + TLS
- [ ] **12** Firebase phone OTP + FCM
- [ ] **13** Maps API key + geocode
- [ ] **14** Global Pay staging webhooks
- [ ] **15** OFD sandbox (only after 10+14)
- [ ] **16** Point apps/portals at staging
- [ ] **17** HPA / observability polish
- [ ] **18** Production promotion (later)
- [ ] **19** Destroy old void-494000 (stop double bill)
