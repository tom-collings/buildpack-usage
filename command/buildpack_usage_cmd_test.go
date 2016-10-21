package command_test

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	"github.com/cloudfoundry/cli/cf/terminal"

	. "github.com/cloudfoundry/cli/cf/i18n"
	. "github.com/ecsteam/buildpack-usage/command"

	"github.com/cloudfoundry/cli/plugin/pluginfakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type testUI struct {
	Input  io.Reader
	Output io.Writer
}

func (ui *testUI) PrintPaginator(rows []string, err error) {}
func (ui *testUI) Say(message string, args ...interface{}) {
	if len(args) == 0 {
		_, _ = fmt.Fprintf(ui.Output, "%s\n", message)
	} else {
		_, _ = fmt.Fprintf(ui.Output, message+"\n", args...)
	}
}

// ProgressReader
func (ui *testUI) PrintCapturingNoOutput(message string, args ...interface{}) {}
func (ui *testUI) Warn(message string, args ...interface{})                   {}
func (ui *testUI) Ask(prompt string) string {
	fmt.Fprintf(ui.Output, "\n%s> ", prompt)

	rd := bufio.NewReader(ui.Input)
	line, err := rd.ReadString('\n')
	if err == nil {
		return strings.TrimSpace(line)
	}
	return ""
}

func (ui *testUI) AskForPassword(prompt string) (answer string)                   { return "" }
func (ui *testUI) Confirm(message string) bool                                    { return false }
func (ui *testUI) ConfirmDelete(modelType, modelName string) bool                 { return false }
func (ui *testUI) ConfirmDeleteWithAssociations(modelType, modelName string) bool { return false }
func (ui *testUI) Ok() {
	ui.Say("OK")
}

func (ui *testUI) Failed(message string, args ...interface{}) {
	ui.Say("FAILED")
	ui.Say(message, args...)
}

func (ui *testUI) ShowConfiguration(coreconfig.Reader) error { return nil }
func (ui *testUI) LoadingIndication()                        {}
func (ui *testUI) NotifyUpdateIfNeeded(coreconfig.Reader)    {}

func (ui testUI) Writer() io.Writer { return ui.Output }

func (ui *testUI) Table(headers []string) *terminal.UITable {
	return &terminal.UITable{
		UI:    ui,
		Table: terminal.NewTable(headers),
	}
}

var _ = Describe("Buildpack-Usage", func() {
	T = func(translationID string, args ...interface{}) string {
		if len(args) == 0 {
			return fmt.Sprintf("%s\n", translationID)
		}

		return fmt.Sprintf(translationID+"\n", args...)
	}

	var fakeCliConnection *pluginfakes.FakeCliConnection

	var convertCommandOutputToStringSlice func(cmd *BuildpackUsage) []string

	BeforeEach(func() {
		fakeCliConnection = &pluginfakes.FakeCliConnection{}
		fakeCliConnection.CliCommandWithoutTerminalOutputStub = func(args ...string) ([]string, error) {
			var output []string

			fixtureName := "fixtures" + args[1] + ".json"

			file, err := os.Open(fixtureName)
			defer file.Close()
			if err != nil {
				Fail("Could not open " + fixtureName)
			}

			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				output = append(output, scanner.Text())
			}

			return output, scanner.Err()
		}

		convertCommandOutputToStringSlice = func(cmd *BuildpackUsage) []string {
			var lines []string
			scanner := bufio.NewScanner(bytes.NewBuffer(cmd.UI.Writer().(*bytes.Buffer).Bytes()))
			for scanner.Scan() {
				line := scanner.Text()
				lines = append(lines, line)
			}

			return lines
		}
	})

	Describe("good buildpack passed on command line", func() {
		var output io.Writer
		var input io.Reader

		var cmd *BuildpackUsage
		BeforeEach(func() {
			output = new(bytes.Buffer)
			input = new(bytes.Buffer)

			ui := &testUI{
				Input:  input,
				Output: output,
			}

			cmd = NewCommand(ui)
		})

		It("returns some apps", func() {
			cmd.Run(fakeCliConnection, []string{"buildpack-usage", "-b", "java_buildpack_offline"})

			lines := convertCommandOutputToStringSlice(cmd)

			Ω(len(lines)).To(Equal(11))
			Ω(lines[2]).To(Equal("OK"))
		})

		It("returns no apps", func() {
			cmd.Run(fakeCliConnection, []string{"buildpack-usage", "-b", "php_buildpack"})
			lines := convertCommandOutputToStringSlice(cmd)

			Ω(lines[2]).To(Equal("OK"))
			Ω(lines[4]).To(Equal("No apps found"))
		})
	})

	Describe("bad buildpack passed on command line", func() {
		var output io.Writer
		var input io.Reader

		var cmd *BuildpackUsage
		BeforeEach(func() {
			output = new(bytes.Buffer)
			input = new(bytes.Buffer)
			ui := &testUI{
				Input:  input,
				Output: output,
			}

			cmd = NewCommand(ui)
		})

		It("fails gracefully", func() {
			buildpackName := "some_dumb_buildpack"
			cmd.Run(fakeCliConnection, []string{"buildpack-usage", "-b", buildpackName})

			lines := convertCommandOutputToStringSlice(cmd)

			Ω(lines[2]).To(Equal("FAILED"))
			Ω(lines[3]).To(Equal(fmt.Sprintf("Error completing request: Could not find buildpack %s", terminal.CommandColor(buildpackName))))
		})
	})

	Describe("good buildpack input on command line", func() {
		var output io.Writer
		var input io.Reader

		var cmd *BuildpackUsage
		BeforeEach(func() {
			output = new(bytes.Buffer)
			input = bytes.NewBufferString("2\n")

			ui := &testUI{
				Input:  input,
				Output: output,
			}

			cmd = NewCommand(ui)
		})

		It("returns some apps", func() {
			cmd.Run(fakeCliConnection, []string{"buildpack-usage"})

			lines := convertCommandOutputToStringSlice(cmd)

			Ω(lines[0]).To(Equal("Please select which buildpack whose apps you would like to see:"))
			Ω(lines[13]).To(Equal("OK"))
			Ω(len(lines[16:])).To(Equal(6))
		})
	})
})
