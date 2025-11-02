package utils

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/fosrl/cli/internal/api"
	"github.com/spf13/viper"
)

// SelectOrg lists organizations for a user and prompts them to select one.
// It returns the selected org ID and any error.
// If the user has only one organization, it's automatically selected.
func SelectOrg(userID string) (string, error) {
	orgsResp, err := api.GlobalClient.ListUserOrgs(userID)
	if err != nil {
		return "", fmt.Errorf("failed to list organizations: %w", err)
	}

	if len(orgsResp.Orgs) == 0 {
		return "", fmt.Errorf("no organizations found for this user")
	}

	if len(orgsResp.Orgs) == 1 {
		// Auto-select if only one org
		selectedOrg := orgsResp.Orgs[0]
		viper.Set("orgId", selectedOrg.OrgID)
		if err := viper.WriteConfig(); err != nil {
			return "", fmt.Errorf("failed to save organization to config: %w", err)
		}
		return selectedOrg.OrgID, nil
	}

	// Multiple orgs - let user select
	type OrgOption struct {
		OrgID string
		Label string
	}

	var orgOptions []huh.Option[OrgOption]
	for _, org := range orgsResp.Orgs {
		label := fmt.Sprintf("%s (%s)", org.Name, org.OrgID)
		orgOptions = append(orgOptions, huh.NewOption(label, OrgOption{
			OrgID: org.OrgID,
			Label: label,
		}))
	}

	var selectedOrgOption OrgOption
	orgSelectForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[OrgOption]().
				Title("Select an organization").
				Options(orgOptions...).
				Value(&selectedOrgOption),
		),
	)

	if err := orgSelectForm.Run(); err != nil {
		return "", fmt.Errorf("error selecting organization: %w", err)
	}

	viper.Set("orgId", selectedOrgOption.OrgID)
	if err := viper.WriteConfig(); err != nil {
		return "", fmt.Errorf("failed to save organization to config: %w", err)
	}

	return selectedOrgOption.OrgID, nil
}
