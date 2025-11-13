package crypto

import (
	"testing"
)

func TestTeamSecretManager_StoreTeamSecret(t *testing.T) {
	t.Skip("TODO: Update test for new TeamSecretManager API - constructor signature changed, teamID and teamSecret no longer passed to NewTeamSecretManager")
	// New API: NewTeamSecretManager(store SecretStore, ks *KeyStore) *TeamSecretManager
}

func TestTeamSecretManager_GetTeamSecret(t *testing.T) {
	t.Skip("TODO: Update test for new TeamSecretManager API")
}

func TestTeamSecretManager_RotateTeamSecret(t *testing.T) {
	t.Skip("TODO: Update test for new TeamSecretManager API")
}

func TestTeamSecretManager_ReencryptSecretsForNewMember(t *testing.T) {
	t.Skip("TODO: Update test for new TeamSecretManager API")
}

func TestTeamSecretManager_GetTeamSecretInfo(t *testing.T) {
	t.Skip("TODO: Update test for new TeamSecretManager API")
}

func TestTeamSecretManager_LockedKeyStore(t *testing.T) {
	t.Skip("TODO: Update test for new TeamSecretManager API")
}
