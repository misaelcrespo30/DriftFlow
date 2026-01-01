package seed

import driftflow "github.com/misaelcrespo30/DriftFlow"

// RegisterSeeders exposes the seeders used by the demo CLI.
func RegisterSeeders() []driftflow.Seeder {
	return []driftflow.Seeder{
		PlanSeeder{},
		TenantSeeder{},
		TenantUserSeeder{},
		UserSeeder{},
	}
}

func init() {
	driftflow.SetSeederRegistry(RegisterSeeders)
}
