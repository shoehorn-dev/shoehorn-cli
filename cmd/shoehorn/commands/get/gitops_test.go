package get

import (
	"reflect"
	"testing"

	"github.com/shoehorn-dev/shoehorn-cli/pkg/api"
)

func TestGitOpsCmd_HasWithoutWorkloadsFlag(t *testing.T) {
	f := gitopsCmd.Flags().Lookup("without-workloads")
	if f == nil {
		t.Fatal("--without-workloads flag is missing")
	}
	if f.Value.Type() != "bool" || f.DefValue != "false" {
		t.Errorf("--without-workloads: type %s default %s, want bool false", f.Value.Type(), f.DefValue)
	}
}

func TestGitOpsListTable(t *testing.T) {
	resources := []*api.GitOpsResource{
		{ClusterID: "prod-eu", Name: "web", Namespace: "argocd", Tool: "argocd", Kind: "Application", SyncStatus: "OutOfSync", HealthStatus: "Missing", LastSyncedAt: "2026-11-26T09:30:00Z",
			NoWorkloads: &api.GitOpsNoWorkloads{Reason: "not_deployed"}},
		{ClusterID: "prod-eu", Name: "jobs", Namespace: "flux-system", Tool: "fluxcd", Kind: "Kustomization", SyncStatus: "NotReady"},
	}

	cols, rows := gitopsListTable(resources, false)
	if !reflect.DeepEqual(cols, []string{"Cluster", "Name", "Namespace", "Tool", "Kind", "Sync Status", "Health Status", "Last Sync"}) {
		t.Errorf("columns without the flag = %v", cols)
	}
	if rows[1][7] != "-" {
		t.Errorf("missing last sync = %q, want -", rows[1][7])
	}

	cols, rows = gitopsListTable(resources, true)
	if cols[len(cols)-1] != "Reason" {
		t.Fatalf("columns with the flag = %v, want Reason last", cols)
	}
	if got := rows[0][len(rows[0])-1]; got != "not_deployed" {
		t.Errorf("reason = %q, want not_deployed", got)
	}
	if got := rows[1][len(rows[1])-1]; got != "-" {
		t.Errorf("reason without no_workloads = %q, want -", got)
	}
}
