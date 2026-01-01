package seed

import driftflow "github.com/misaelcrespo30/DriftFlow"

func RegisterSeeders() []driftflow.Seeder {
	return []driftflow.Seeder{
		UserSeeder{},
		PlanSeeder{},
		TenantSeeder{},
		TenantUserSeeder{},
	}
}
