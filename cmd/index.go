// Copyright © 2026 Pavel Dimens | Github: pdimens
package cmd

import (
	"arachne/gominibwa"
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var indexCmd = &cobra.Command{
	Short:   "Index reference.fasta",
	Use:     "index REF.fa",
	Example: "arachne index ref.fa",
	Long: "A small wrapper to use minibwa (included) to index a reference FASTA file prior to alignment. " +
		"This creates REF.fa.l2b and REF.fa.mbw alongside the FASTA.",
	DisableFlagsInUseLine: true,
	SilenceUsage:          true,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			fmt.Printf("%s", cmd.UsageString())
			return fmt.Errorf("please provide input reference FASTA")
		}
		if err := cobra.ExactArgs(1)(cmd, args); err != nil {
			return err
		}
		if err := filecheck(args[0]); err != nil {
			return err
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		threads, _ := cmd.Flags().GetInt("threads")
		verbose, _ := cmd.Flags().GetBool("verbose")
		runtime.GOMAXPROCS(safethreads(threads))
		return minibwaIndex(args[0], threads, verbose)
	},
}

func init() {
	rootCmd.AddCommand(indexCmd)
	indexCmd.Flags().IntP("threads", "t", 4, "Threads to use")
	indexCmd.Flags().BoolP("verbose", "v", false, "Verbose output logging")
}

// Thin shim/wrapper to index a reference with minibwa index. Stderr only
// prints on error.
func minibwaIndex(fasta string, threads int, verbose bool) error {
	if verbose {
		fmt.Printf("   Building minibwa index for %s...\n", fasta)
	}
	err := gominibwa.Build(fasta, threads)
	if err != nil {
		return fmt.Errorf("Failed to build minibwa reference index.\n%v", err)
	}
	if verbose {
		fmt.Println("   Index built successfully!")
	}
	return nil
}
