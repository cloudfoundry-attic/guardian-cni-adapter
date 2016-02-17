package acceptance_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

type fakePluginLogData struct {
	Args  []string
	Env   map[string]string
	Stdin string
}

func getConfig(index int) string {
	return fmt.Sprintf(`
{
  "cniVersion": "0.1.0",
  "name": "some-net-%d",
  "type": "plugin-%d"
}`, index, index)
}

func writeConfig(index int, outDir string) error {
	config := getConfig(index)
	outpath := filepath.Join(outDir, fmt.Sprintf("%d-plugin-%d.conf", 10*index, index))
	return ioutil.WriteFile(outpath, []byte(config), 0600)
}

var _ = Describe("Guardian CNI adapter", func() {
	var (
		command           *exec.Cmd
		cniConfigDir      string
		fakePid           int
		fakeLogDir        string
		expectedNetNSPath string
	)

	BeforeEach(func() {
		var err error
		cniConfigDir, err = ioutil.TempDir("", "cni-config-")
		Expect(err).NotTo(HaveOccurred())

		fakeLogDir, err = ioutil.TempDir("", "fake-logs-")
		Expect(err).NotTo(HaveOccurred())

		command = exec.Command(pathToAdapter)
		command.Env = []string{"FAKE_LOG_DIR=" + fakeLogDir}

		fakePid = rand.Intn(30000)
		command.Stdin = strings.NewReader(fmt.Sprintf(`{ "pid": %d }`, fakePid))

		expectedNetNSPath = fmt.Sprintf("/proc/%d/ns/net", fakePid)

		Expect(writeConfig(0, cniConfigDir)).To(Succeed())
		Expect(writeConfig(1, cniConfigDir)).To(Succeed())
		Expect(writeConfig(2, cniConfigDir)).To(Succeed())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(cniConfigDir)).To(Succeed())
		Expect(os.RemoveAll(fakeLogDir)).To(Succeed())
	})

	Describe("up", func() {
		BeforeEach(func() {
			command.Args = []string{pathToAdapter,
				"--cniPluginDir", cniPluginDir,
				"--cniConfigDir", cniConfigDir,
				"--ducatiSandboxDir", "some-sandbox",
				"--daemonBaseURL", "http://example.com",
				"up",
				"--handle", "some-container-handle",
				"--network", "some-network-spec",
			}
		})

		It("should call every CNI plugin in the plugin directory with ADD", func() {
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))

			for i := 0; i < 3; i++ {
				logFileContents, err := ioutil.ReadFile(filepath.Join(fakeLogDir, fmt.Sprintf("plugin-%d.log", i)))
				Expect(err).NotTo(HaveOccurred())
				var pluginCallInfo fakePluginLogData
				Expect(json.Unmarshal(logFileContents, &pluginCallInfo)).To(Succeed())

				Expect(pluginCallInfo.Stdin).To(MatchJSON(getConfig(i)))
				Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_COMMAND", "ADD"))
				Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_CONTAINERID", "some-container-handle"))
				Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_IFNAME", fmt.Sprintf("eth%d", i)))
				Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_PATH", cniPluginDir))
				Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_NETNS", expectedNetNSPath))
				Expect(pluginCallInfo.Env).To(HaveKeyWithValue("DUCATI_OS_SANDBOX_REPO", "some-sandbox"))
				Expect(pluginCallInfo.Env).To(HaveKeyWithValue("DAEMON_BASE_URL", "http://example.com"))
				Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_ARGS", ""))
			}
		})
	})

	Describe("down", func() {
		BeforeEach(func() {
			command.Args = []string{pathToAdapter,
				"down",
				"--handle", "some-container-handle",
				"--cniPluginDir", cniPluginDir,
				"--cniConfigDir", cniConfigDir,
				"--ducatiSandboxDir", "some-sandbox",
				"--daemonBaseURL", "http://example.com",
			}
		})

		It("should call every CNI plugin in the plugin directory with DEL", func() {
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))

			for i := 0; i < 3; i++ {
				logFileContents, err := ioutil.ReadFile(filepath.Join(fakeLogDir, fmt.Sprintf("plugin-%d.log", i)))
				Expect(err).NotTo(HaveOccurred())
				var pluginCallInfo fakePluginLogData
				Expect(json.Unmarshal(logFileContents, &pluginCallInfo)).To(Succeed())

				Expect(pluginCallInfo.Stdin).To(MatchJSON(getConfig(i)))
				Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_COMMAND", "DEL"))
				Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_CONTAINERID", "some-container-handle"))
				Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_IFNAME", fmt.Sprintf("eth%d", i)))
				Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_PATH", cniPluginDir))
				Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_NETNS", expectedNetNSPath))
				Expect(pluginCallInfo.Env).To(HaveKeyWithValue("DUCATI_OS_SANDBOX_REPO", "some-sandbox"))
				Expect(pluginCallInfo.Env).To(HaveKeyWithValue("DAEMON_BASE_URL", "http://example.com"))
				Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_ARGS", ""))
			}
		})
	})
})
