package deploy

import (
	"context"
	"fmt"
	"os/exec"
)

// reconcileFluxSource triggers flux reconciliation for a GitRepository source
func (r *runner) reconcileFluxSource(ctx context.Context, resourceName, namespace string) error {
	if !r.flag.Sync {
		return nil
	}

	// Check if flux CLI is available
	if _, err := exec.LookPath("flux"); err != nil {
		return fmt.Errorf("flux CLI not found in PATH. Please install flux CLI to use --sync flag: %w", err)
	}

	//nolint:errcheck // informational output
	fmt.Fprintf(r.stdout, "%s Reconciling flux source %s...\n", infoStyle.Render("→"), resourceName)

	// Run: flux reconcile source git <name> -n <namespace>
	cmd := exec.CommandContext(ctx, "flux", "reconcile", "source", "git", resourceName, "-n", namespace)
	cmd.Stdout = r.stdout
	cmd.Stderr = r.stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reconcile flux source: %w", err)
	}

	return nil
}

// reconcileFluxApp triggers flux reconciliation for app-related kustomizations
// Apps deployed with this tool have flux reconciliation suspended, so we reconcile
// the kustomizations in the flux-giantswarm namespace to ensure cluster-level changes are applied
func (r *runner) reconcileFluxApp(ctx context.Context, appName, namespace string) error {
	if !r.flag.Sync {
		return nil
	}

	// Check if flux CLI is available
	if _, err := exec.LookPath("flux"); err != nil {
		return fmt.Errorf("flux CLI not found in PATH. Please install flux CLI to use --sync flag: %w", err)
	}

	//nolint:errcheck // informational output
	fmt.Fprintf(r.stdout, "%s Reconciling flux kustomizations...\n", infoStyle.Render("→"))

	// Reconcile the management-clusters-fleet kustomization which manages apps
	cmd := exec.CommandContext(ctx, "flux", "reconcile", "kustomization", "collection", "-n", "flux-giantswarm")
	cmd.Stdout = r.stdout
	cmd.Stderr = r.stderr

	if err := cmd.Run(); err != nil {
		// If management-clusters-fleet doesn't exist, that's okay - just warn
		//nolint:errcheck // non-critical warning message
		fmt.Fprintf(r.stderr, "Warning: failed to reconcile flux kustomizations: %v\n", err)
		return nil
	}

	return nil
}
