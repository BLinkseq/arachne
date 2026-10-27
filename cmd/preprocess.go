// Copyright © 2026 Pavel Dimens | Github: pdimens
package cmd

import (
	"fmt"
	"runtime"

	"github.com/blinkseq/djinn/sort"

	"github.com/spf13/cobra"
)

var prep = &cobra.Command{
	Use:     "prep [options] PREFIX R1.fq R2.fq",
	Short:   "Standardize FASTQ files and sort by BX barcode",
	Example: "prep -t 12 sample1 smp1.R1.fq.gz smp1.R2.fq.gz",
	Long: "Convert a pair of linked-read FASTQ files into standard format and sort by barcode, as required by arachne aligner. " +
		"Include barcode list (or literal '10x' for embedded official 10X barcode list) if input is 10X data. Output is PREFIX.arachne.R[1|2].fq.gz.",
	DisableFlagsInUseLine: true,
	SilenceUsage:          true,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			fmt.Printf("%s", cmd.UsageString())
			return fmt.Errorf("please provide inputs")
		}
		if err := cobra.ExactArgs(3)(cmd, args); err != nil {
			return err
		}
		if err := filecheck(args[1]); err != nil {
			return err
		}
		if err := filecheck(args[2]); err != nil {
			return err
		}
		if _, err := cmd.Flags().GetInt("threads"); err != nil {
			return err
		}

		bcMap, err := cmd.Flags().GetString("bc-list")
		if err != nil {
			return err
		}
		switch bcMap {
		case "", "10x", "10X": // skip check
		default:
			if err := filecheck(bcMap); err != nil {
				return err
			}
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		threads, _ := cmd.Flags().GetInt("threads")
		runtime.GOMAXPROCS(safethreads(threads))

		tmpDir, _ := cmd.Flags().GetString("tmp-prefix")
		bcMap, _ := cmd.Flags().GetString("bc-list")

		return sort.SortFqByBX(args[1:], args[0], tmpDir, threads, true, bcMap)
	},
}

func init() {
	rootCmd.AddCommand(prep)
	prep.Flags().IntP("threads", "t", 2, "Threads to use")
	prep.Flags().StringP("tmp-prefix", "p", "", "Folder for temporary files")
	prep.Flags().StringP("bc-list", "b", "", "Valid barcode list (10X Genomics input only)")
}
