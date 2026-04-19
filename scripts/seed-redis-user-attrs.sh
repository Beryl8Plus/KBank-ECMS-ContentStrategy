#!/usr/bin/env sh
# Seed Redis with sample CIS user attributes for local development / testing.
#
# Key format  : cis_id:{cisID}
# Value format: JSON object keyed by attribute UUIDs defined in the mock seed data.
#
# Attribute UUIDs (from 20260417100237_decision_rule_example_data.sql):
#   segment    b48e1fce-d061-4d47-9845-6b9022e19361  Text   Mass | Affluent | VIP | Young Wealth | SME
#   user_age   9953ad6b-f683-4ae4-a93d-b4dba619d109  Number
#   region     5dae579a-be32-4a5a-afa4-a07846b7e21c  Text   Bangkok | Central | North | Northeast | South
#   risk_level 6678c712-f299-437c-b818-dd4d275a0681  Number 1 | 2 | 3 | 4 | 5

REDIS_CLI="${REDIS_CLI:-redis-cli}"

set -e

seed() {
  CIS_ID="$1"
  PAYLOAD="$2"
  $REDIS_CLI SET "cis_id:${CIS_ID}" "${PAYLOAD}"
  echo "  SET cis_id:${CIS_ID}"
}

echo "Seeding Redis user attributes..."

# Mass segment — Bangkok — age 28 — risk 2
seed "cis-user-01" '{"b48e1fce-d061-4d47-9845-6b9022e19361":"Mass","9953ad6b-f683-4ae4-a93d-b4dba619d109":28,"5dae579a-be32-4a5a-afa4-a07846b7e21c":"Bangkok","6678c712-f299-437c-b818-dd4d275a0681":2}'

# Affluent segment — Central — age 35 — risk 3
seed "cis-user-02" '{"b48e1fce-d061-4d47-9845-6b9022e19361":"Affluent","9953ad6b-f683-4ae4-a93d-b4dba619d109":35,"5dae579a-be32-4a5a-afa4-a07846b7e21c":"Central","6678c712-f299-437c-b818-dd4d275a0681":3}'

# VIP segment — North — age 45 — risk 4
seed "cis-user-03" '{"b48e1fce-d061-4d47-9845-6b9022e19361":"VIP","9953ad6b-f683-4ae4-a93d-b4dba619d109":45,"5dae579a-be32-4a5a-afa4-a07846b7e21c":"North","6678c712-f299-437c-b818-dd4d275a0681":4}'

# Young Wealth segment — Northeast — age 24 — risk 1
seed "cis-user-04" '{"b48e1fce-d061-4d47-9845-6b9022e19361":"Young Wealth","9953ad6b-f683-4ae4-a93d-b4dba619d109":24,"5dae579a-be32-4a5a-afa4-a07846b7e21c":"Northeast","6678c712-f299-437c-b818-dd4d275a0681":1}'

# SME segment — South — age 52 — risk 5
seed "cis-user-05" '{"b48e1fce-d061-4d47-9845-6b9022e19361":"SME","9953ad6b-f683-4ae4-a93d-b4dba619d109":52,"5dae579a-be32-4a5a-afa4-a07846b7e21c":"South","6678c712-f299-437c-b818-dd4d275a0681":5}'

# Mass segment — Northeast — age 60 — risk 2  (triggers age >= 56 and region = Northeast rules)
seed "cis-user-06" '{"b48e1fce-d061-4d47-9845-6b9022e19361":"Mass","9953ad6b-f683-4ae4-a93d-b4dba619d109":60,"5dae579a-be32-4a5a-afa4-a07846b7e21c":"Northeast","6678c712-f299-437c-b818-dd4d275a0681":2}'

# Affluent segment — South — age 30 — risk 4  (triggers region = South and risk >= 4 rules)
seed "cis-user-07" '{"b48e1fce-d061-4d47-9845-6b9022e19361":"Affluent","9953ad6b-f683-4ae4-a93d-b4dba619d109":30,"5dae579a-be32-4a5a-afa4-a07846b7e21c":"South","6678c712-f299-437c-b818-dd4d275a0681":4}'

# VIP segment — Bangkok — age 38 — risk 5  (max risk)
seed "cis-user-08" '{"b48e1fce-d061-4d47-9845-6b9022e19361":"VIP","9953ad6b-f683-4ae4-a93d-b4dba619d109":38,"5dae579a-be32-4a5a-afa4-a07846b7e21c":"Bangkok","6678c712-f299-437c-b818-dd4d275a0681":5}'

# Young Wealth segment — Central — age 22 — risk 1  (low age, low risk)
seed "cis-user-09" '{"b48e1fce-d061-4d47-9845-6b9022e19361":"Young Wealth","9953ad6b-f683-4ae4-a93d-b4dba619d109":22,"5dae579a-be32-4a5a-afa4-a07846b7e21c":"Central","6678c712-f299-437c-b818-dd4d275a0681":1}'

# SME segment — North — age 48 — risk 3
seed "cis-user-10" '{"b48e1fce-d061-4d47-9845-6b9022e19361":"SME","9953ad6b-f683-4ae4-a93d-b4dba619d109":48,"5dae579a-be32-4a5a-afa4-a07846b7e21c":"North","6678c712-f299-437c-b818-dd4d275a0681":3}'

echo "Done. Seeded 10 CIS user attribute records."
