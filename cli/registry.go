package cli

import "github.com/spf13/cobra"

var Commands = []*cobra.Command{
	newUpCommand(),
	newDownCommand(),
	newUndoCommand(),
	newSeedCommand(),
	newSeedgenCommand(),
	newGenerateCommand(),
	newMigrateCommand(),
	newValidateCommand(),
	newAuditCommand(),
	newCompareCommand(),
}
