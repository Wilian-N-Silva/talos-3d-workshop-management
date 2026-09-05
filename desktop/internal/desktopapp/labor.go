package desktopapp

import "github.com/Wilian-N-Silva/talos-3d-workshop-management/desktop/internal/apiclient"

func (a *App) ListLaborRates() ([]apiclient.LaborRate, error) {
	client, session, baseURL, err := a.authenticatedClient()
	if err != nil {
		return nil, err
	}
	value, err := client.ListLaborRates(a.applicationContext(), session.Token)
	if err != nil {
		return nil, a.handleAuthenticatedError(baseURL, err)
	}
	return value, nil
}
func (a *App) SaveLaborRate(id string, input apiclient.LaborRateInput) (apiclient.LaborRate, error) {
	client, session, baseURL, err := a.authenticatedClient()
	if err != nil {
		return apiclient.LaborRate{}, err
	}
	value, err := client.SaveLaborRate(a.applicationContext(), session.Token, id, input)
	if err != nil {
		return apiclient.LaborRate{}, a.handleAuthenticatedError(baseURL, err)
	}
	return value, nil
}
func (a *App) SuggestLaborRate(input apiclient.LaborAssumptions) (apiclient.LaborSuggestion, error) {
	client, session, baseURL, err := a.authenticatedClient()
	if err != nil {
		return apiclient.LaborSuggestion{}, err
	}
	value, err := client.SuggestLaborRate(a.applicationContext(), session.Token, input)
	if err != nil {
		return apiclient.LaborSuggestion{}, a.handleAuthenticatedError(baseURL, err)
	}
	return value, nil
}
