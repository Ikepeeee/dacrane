package cmd

import (
	"dacrane/core"
	"dacrane/utils"
	"os"

	"github.com/spf13/cobra"
)

// publishCmd represents the publish command
var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish the specific artifact",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		codeBytes, err := os.ReadFile("dacrane.yaml")
		if err != nil {
			panic(err)
		}

		codes, err := core.ParseCode(codeBytes)
		if err != nil {
			panic(err)
		}

		artifactCode := utils.Find(codes, func(code core.Code) bool {
			return code.Kind == "artifact" && code.Name == name
		})

		artifactProvider := core.FindArtifactProvider(artifactCode.Provider)

		result, err := artifactProvider.Publish(artifactCode.Parameters)

		if err != nil {
			println(string(result))
			panic(err)
		}
		println("publish successfully!")
	},
}

func init() {
	rootCmd.AddCommand(publishCmd)

}
