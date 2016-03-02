package acceptance_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
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

func sameFile(path1, path2 string) bool {
	fi1, err := os.Stat(path1)
	Expect(err).NotTo(HaveOccurred())

	fi2, err := os.Stat(path2)
	Expect(err).NotTo(HaveOccurred())
	return os.SameFile(fi1, fi2)
}

const DEFAULT_TIMEOUT = "5s"

var _ = Describe("Guardian CNI adapter", func() {
	var (
		cniConfigDir      string
		fakePid           int
		fakeLogDir        string
		expectedNetNSPath string
		bindMountRoot     string
		containerHandle   string
		fakeProcess       *os.Process
	)

	BeforeEach(func() {
		var err error
		cniConfigDir, err = ioutil.TempDir("", "cni-config-")
		Expect(err).NotTo(HaveOccurred())

		fakeLogDir, err = ioutil.TempDir("", "fake-logs-")
		Expect(err).NotTo(HaveOccurred())

		containerHandle = "some-container-handle"

		sleepCmd := exec.Command("/bin/sleep", "1000")
		Expect(sleepCmd.Start()).To(Succeed())
		fakeProcess = sleepCmd.Process

		fakePid = fakeProcess.Pid

		bindMountRoot, err = ioutil.TempDir("", "bind-mount-root")
		Expect(err).NotTo(HaveOccurred())

		expectedNetNSPath = fmt.Sprintf("%s/%s", bindMountRoot, containerHandle)

		Expect(writeConfig(0, cniConfigDir)).To(Succeed())
		Expect(writeConfig(1, cniConfigDir)).To(Succeed())
		Expect(writeConfig(2, cniConfigDir)).To(Succeed())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(cniConfigDir)).To(Succeed())
		Expect(os.RemoveAll(fakeLogDir)).To(Succeed())
		Expect(fakeProcess.Kill()).To(Succeed())
	})

	Describe("CNI plugin lifecycle events", func() {
		It("should call CNI ADD and DEL", func() {

			By("calling up")
			upCommand := exec.Command(pathToAdapter)
			upCommand.Env = []string{"FAKE_LOG_DIR=" + fakeLogDir}
			upCommand.Stdin = strings.NewReader(fmt.Sprintf(`{ "pid": %d }`, fakePid))
			upCommand.Args = []string{pathToAdapter,
				"--cniPluginDir", cniPluginDir,
				"--cniConfigDir", cniConfigDir,
				"--nsBindMountRoot", bindMountRoot,
				"--action", "up",
				"--handle", "some-container-handle",
				"--network", "some-network-spec",
			}

			upSession, err := gexec.Start(upCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(upSession, DEFAULT_TIMEOUT).Should(gexec.Exit(0))

			By("checking that every CNI plugin in the plugin directory got called with ADD")
			for i := 0; i < 3; i++ {
				logFileContents, err := ioutil.ReadFile(filepath.Join(fakeLogDir, fmt.Sprintf("plugin-%d.log", i)))
				Expect(err).NotTo(HaveOccurred())
				var pluginCallInfo fakePluginLogData
				Expect(json.Unmarshal(logFileContents, &pluginCallInfo)).To(Succeed())

				Expect(pluginCallInfo.Stdin).To(MatchJSON(getConfig(i)))
				Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_COMMAND", "ADD"))
				Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_CONTAINERID", containerHandle))
				Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_IFNAME", fmt.Sprintf("eth%d", i)))
				Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_PATH", cniPluginDir))
				Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_NETNS", expectedNetNSPath))
				Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_ARGS", ""))
			}

			By("checking that the fake process's network namespace has been bind-mounted into the filesystem")
			Expect(sameFile(expectedNetNSPath, fmt.Sprintf("/proc/%d/ns/net", fakePid))).To(BeTrue())

			By("calling down")
			downCommand := exec.Command(pathToAdapter)
			downCommand.Env = []string{"FAKE_LOG_DIR=" + fakeLogDir}
			downCommand.Stdin = strings.NewReader(`{}`)
			downCommand.Args = []string{pathToAdapter,
				"--action", "down",
				"--handle", "some-container-handle",
				"--cniPluginDir", cniPluginDir,
				"--cniConfigDir", cniConfigDir,
				"--nsBindMountRoot", bindMountRoot,
			}

			downSession, err := gexec.Start(downCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(downSession, DEFAULT_TIMEOUT).Should(gexec.Exit(0))

			By("checking that every CNI plugin in the plugin directory got called with DEL")
			for i := 0; i < 3; i++ {
				logFileContents, err := ioutil.ReadFile(filepath.Join(fakeLogDir, fmt.Sprintf("plugin-%d.log", i)))
				Expect(err).NotTo(HaveOccurred())
				var pluginCallInfo fakePluginLogData
				Expect(json.Unmarshal(logFileContents, &pluginCallInfo)).To(Succeed())

				Expect(pluginCallInfo.Stdin).To(MatchJSON(getConfig(i)))
				Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_COMMAND", "DEL"))
				Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_CONTAINERID", containerHandle))
				Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_IFNAME", fmt.Sprintf("eth%d", i)))
				Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_PATH", cniPluginDir))
				Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_NETNS", expectedNetNSPath))
				Expect(pluginCallInfo.Env).To(HaveKeyWithValue("CNI_ARGS", ""))
			}

			By("checking that the bind-mounted namespace has been removed")
			Expect(expectedNetNSPath).NotTo(BeAnExistingFile())
		})
	})
})
