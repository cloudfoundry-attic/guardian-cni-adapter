package acceptance_test

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Guardian CNI adapter", func() {
	var (
		command      *exec.Cmd
		cniConfigDir string
		fakePid      int
		fakeLogDir   string
	)

	BeforeEach(func() {
		var err error
		cniConfigDir, err = ioutil.TempDir("", "cni-config-")
		Expect(err).NotTo(HaveOccurred())

		fakeLogDir, err = ioutil.TempDir("", "fake-logs-")
		Expect(err).NotTo(HaveOccurred())

		bindMountRoot, err := ioutil.TempDir("", "bind-mount-root")
		Expect(err).NotTo(HaveOccurred())

		command = exec.Command(pathToAdapter)
		command.Args = []string{pathToAdapter,
			"up",
			"--handle=some-container-handle",
			"--network=some-network-spec",
			"--cniPluginDir=" + cniPluginDir,
			"--cniConfigDir=" + cniConfigDir,
			"--ducatiSandboxDir=" + "some-sandbox-dir",
			"--daemonBaseURL=" + "http://example.com",
			"--nsBindMountRoot=" + bindMountRoot,
		}
		command.Env = []string{"FAKE_LOG_DIR=" + fakeLogDir}

		fakePid = rand.Intn(30000)
		command.Stdin = strings.NewReader(fmt.Sprintf(`{ "pid": %d }`, fakePid))

		Expect(writeConfig(0, cniConfigDir)).To(Succeed())
		Expect(writeConfig(1, cniConfigDir)).To(Succeed())
		Expect(writeConfig(2, cniConfigDir)).To(Succeed())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(cniConfigDir)).To(Succeed())
		Expect(os.RemoveAll(fakeLogDir)).To(Succeed())
	})

	Context("when the config files reference a plugin that isn't available", func() {
		XIt("should exit status 1 and print a useful error", func() {

		})
	})

	Context("when inputs are invalid", func() {
		Context("when stdin is not valid JSON", func() {
			It("should exit status 1 and print an error to stderr", func() {
				command.Stdin = strings.NewReader("{{{bad")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Out.Contents()).To(BeEmpty())
				Expect(session.Err.Contents()).To(ContainSubstring("json"))
				Expect(session.Err.Contents()).To(ContainSubstring("{{{bad"))
			})
		})

		Context("when the stdin JSON is missing a pid field", func() {
			It("should exit status 1 and print an error to stderr", func() {
				command.Stdin = strings.NewReader(`{ "something": 12 }`)
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Out.Contents()).To(BeEmpty())
				Expect(session.Err.Contents()).To(ContainSubstring("missing pid"))
			})
		})

		Context("when the provided pid is not an integer", func() {
			It("should exit status 1 and print an error to stderr", func() {
				command.Stdin = strings.NewReader(`{ "pid": "not-a-number"  }`)
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Out.Contents()).To(BeEmpty())
				Expect(session.Err.Contents()).To(ContainSubstring(`cannot unmarshal string into Go value of type int`))
			})
		})

		Context("when the action is incorrect", func() {
			It("should return an error", func() {
				command.Args[1] = "some-invalid-action"

				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Out.Contents()).To(BeEmpty())
				Expect(session.Err.Contents()).To(ContainSubstring(`action: some-invalid-action is unrecognized`))
			})
		})

		Context("when an unknown flag is provided", func() {
			It("should return an error", func() {
				command.Args = append(command.Args, "--banana")

				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Out.Contents()).To(BeEmpty())
				Expect(session.Err.Contents()).To(ContainSubstring(`flag provided but not defined: -banana`))
			})
		})

		Context("when an unknown positional arg is provided", func() {
			It("should return an error", func() {
				command.Args = append(command.Args, "something-else")

				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Out.Contents()).To(BeEmpty())
				Expect(session.Err.Contents()).To(ContainSubstring(`unexpected extra args: [something-else]`))
			})
		})

		var removeArrayElement = func(src []string, elementToRemove string) []string {
			reduced := []string{}
			for _, element := range src {
				if !strings.HasPrefix(element, elementToRemove) {
					reduced = append(reduced, element)
				}
			}

			return reduced
		}

		DescribeTable("missing required arguments",
			func(missingFlag string) {
				command.Args = removeArrayElement(command.Args, "--"+missingFlag)

				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Out.Contents()).To(BeEmpty())
				Expect(session.Err.Contents()).To(ContainSubstring(
					fmt.Sprintf("missing required flag '%s'", missingFlag)))
			},
			Entry("handle", "handle"),
			Entry("network", "network"),
			Entry("cniPluginDir", "cniPluginDir"),
			Entry("cniConfigDir", "cniConfigDir"),
			Entry("ducatiSandboxDir", "ducatiSandboxDir"),
			Entry("daemonBaseURL", "daemonBaseURL"),
			Entry("nsBindMountRoot", "nsBindMountRoot"),
		)

		Context("when the user doesn't know what to do", func() {
			DescribeTable("arguments that indicate ignorance",
				func(args []string) {
					command.Args = args
					command.Stdin = strings.NewReader("invalid json")

					session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())

					Eventually(session).Should(gexec.Exit(1))
					Expect(session.Out.Contents()).To(BeEmpty())
					Expect(session.Err.Contents()).To(ContainSubstring(`this is a OCI prestart/poststop hook.  see https://github.com/opencontainers/specs/blob/master/runtime-config.md`))
				},
				Entry("no args", []string{pathToAdapter}),
				Entry("short help", []string{pathToAdapter, "-h"}),
				Entry("long help", []string{pathToAdapter, "--help"}),
			)
		})
	})
})
