ทำ CI/CD Deploy ไปที่ GKE

# สร้าง SA
gcloud iam service-accounts create github-actions-deployer \
  --project=be8-ecms-test \
  --display-name="GitHub Actions Deployer"

# ให้ push image ได้
gcloud projects add-iam-policy-binding be8-ecms-test \
  --member="serviceAccount:github-actions-deployer@be8-ecms-test.iam.gserviceaccount.com" \
  --role="roles/artifactregistry.writer"

# ให้ kubectl rollout restart ได้
gcloud projects add-iam-policy-binding be8-ecms-test \
  --member="serviceAccount:github-actions-deployer@be8-ecms-test.iam.gserviceaccount.com" \
  --role="roles/container.developer"

# Export JSON key
gcloud iam service-accounts keys create sa-key.json \
  --iam-account=github-actions-deployer@be8-ecms-test.iam.gserviceaccount.com
