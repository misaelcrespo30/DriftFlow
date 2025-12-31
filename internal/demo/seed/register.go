package seed

import driftflow "github.com/misaelcrespo30/DriftFlow"

func RegisterSeeders() []driftflow.Seeder {
	return []driftflow.Seeder{
		// Orden de ejecución: primero tablas base, luego relaciones.
		PlanSeeder{},
		TenantSeeder{},
		UserSeeder{},
		TenantUserSeeder{},
	}
}
