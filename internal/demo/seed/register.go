package seed

import driftflow "github.com/misaelcrespo30/DriftFlow"

func RegisterSeeders() []driftflow.Seeder {
	return []driftflow.Seeder{
		PlanSeeder{},
		TenantSeeder{},
		UserSeeder{},
		TenantUserSeeder{},
	}
}
