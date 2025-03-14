package state

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseStateFile(t *testing.T) {
	for _, c := range []struct {
		name     string
		filename string
		want     V4
	}{{
		name:     "example.tfstate",
		filename: "testdata/example.tfstate",
		want: V4{
			Version: 4,
			Resources: []Resource{{
				Mode:     "data",
				Type:     "chainguard_role",
				Name:     "roles",
				Provider: "provider[\"registry.terraform.io/chainguard-dev/chainguard\"]",
				Instances: []Instance{{
					IndexKey: "registry.pull",
					Attributes: map[string]interface{}{
						"id": "registry.pull",
					},
				}, {
					IndexKey: "registry.push",
					Attributes: map[string]interface{}{
						"id": "registry.push",
					},
				}},
			}, {
				Mode:     "managed",
				Type:     "chainguard_group",
				Name:     "group",
				Provider: "provider[\"registry.terraform.io/chainguard-dev/chainguard\"]",
				Instances: []Instance{{
					Attributes: map[string]interface{}{
						"id": "1350aba7a5c3f0a6e6183b1f8b16fe563bff5150",
					},
				}},
			}, {
				Mode:     "managed",
				Type:     "chainguard_group_invite",
				Name:     "invite-code",
				Provider: "provider[\"registry.terraform.io/chainguard-dev/chainguard\"]",
				Instances: []Instance{{
					Attributes: map[string]interface{}{
						"id": "1350aba7a5c3f0a6e6183b1f8b16fe563bff5150/4d004baae5b84eb4",
					},
					Dependencies: []string{
						"chainguard_group.group",
						"data.chainguard_role.roles",
					},
				}},
			}, {
				Mode:     "managed",
				Type:     "chainguard_identity",
				Name:     "assumed-identity",
				Provider: "provider[\"registry.terraform.io/chainguard-dev/chainguard\"]",
				Instances: []Instance{{
					IndexKey: "api",
					Attributes: map[string]interface{}{
						"id": "1350aba7a5c3f0a6e6183b1f8b16fe563bff5150/10922bb064b4f6ff",
					},
					Dependencies: []string{
						"chainguard_group.group",
					},
				}, {
					IndexKey: "build",
					Attributes: map[string]interface{}{
						"id": "1350aba7a5c3f0a6e6183b1f8b16fe563bff5150/e2f2ce0808ae0d7b",
					},
					Dependencies: []string{
						"chainguard_group.group",
					},
				}},
			}, {
				Mode:     "managed",
				Module:   "module.api",
				Type:     "google_monitoring_alert_policy",
				Name:     "alert",
				Provider: "provider[\"registry.terraform.io/hashicorp/google\"]",
				Instances: []Instance{{
					// json.Unmarshal defaults to float64 for json numbers
					IndexKey: float64(0),
					Attributes: map[string]interface{}{
						"id": "projects/prod/alertPolicies/3257823384035534535",
					},
					Dependencies: []string{
						"chainguard_group.group",
						"chainguard_identity.assumed-identity",
						"module.api.module.this.module.this.google_project_iam_member.metrics-writer",
					},
				}},
			}, {
				Mode:     "managed",
				Module:   "module.api.module.gclb[0]",
				Type:     "google_compute_backend_service",
				Name:     "public-services",
				Provider: "provider[\"registry.terraform.io/hashicorp/google\"]",
				Instances: []Instance{{
					IndexKey: "api",
					Attributes: map[string]interface{}{
						"id": "projects/prod/global/backendServices/api",
					},
				}},
			}, {
				Mode:     "managed",
				Module:   "module.api.module.this.module.this",
				Type:     "google_project_iam_member",
				Name:     "metrics-writer",
				Provider: "provider[\"registry.terraform.io/hashicorp/google\"]",
				Instances: []Instance{{
					Attributes: map[string]interface{}{
						"condition": []any{},
						"id":        "prod/roles/monitoring.metricWriter/serviceAccount:api@prod.iam.gserviceaccount.com",
						"member":    "serviceAccount:api@prod.iam.gserviceaccount.com",
						"project":   "prod",
						"role":      "roles/monitoring.metricWriter",
					},
				}},
			}, {
				Mode:     "managed",
				Module:   "module.api.module.this.module.this",
				Type:     "google_cloud_run_v2_service_iam_member",
				Name:     "public-services-are-unauthenticated",
				Provider: "provider[\"registry.terraform.io/hashicorp/google\"]",
				Instances: []Instance{{
					IndexKey: "us-central1",
					Attributes: map[string]interface{}{
						"condition": []any{},
						"id":        "projects/prod/locations/us-central1/services/api/roles/run.invoker/allUsers",
						"location":  "us-central1",
						"member":    "allUsers",
						"name":      "projects/prod/locations/us-central1/services/api",
						"project":   "prod",
						"role":      "roles/run.invoker",
					},
					Dependencies: []string{
						"chainguard_group.group",
						"chainguard_identity.assumed-identity",
					},
				}},
			}},
		},
	}} {
		t.Run(c.name, func(t *testing.T) {
			got, err := ParseStateFile(c.filename)
			if err != nil {
				t.Fatalf("failed to parse statefile \"testdata/example.tfstate\": %v", err)
			}
			if diff := cmp.Diff(c.want, got); diff != "" {
				t.Errorf("ParseStateFile() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
