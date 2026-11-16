package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"text/template"
)

const label = "local.android-mtp"

var plistTmpl = template.Must(template.New("plist").Funcs(template.FuncMap{"xml": xmlEscape}).Parse(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>{{xml .Label}}</string>
	<key>ProgramArguments</key>
	<array>
		<string>{{xml .Bin}}</string>
		<string>watch</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
	<key>ProcessType</key>
	<string>Interactive</string>
	<key>StandardOutPath</key>
	<string>{{xml .Log}}</string>
	<key>StandardErrorPath</key>
	<string>{{xml .Log}}</string>
</dict>
</plist>
`))

// agent décrit l'agent launchd et l'emplacement de ses fichiers.
type agent struct {
	Label, Bin, Log, Plist string
}

func newAgent(home string) agent {
	return agent{
		Label: label,
		Bin:   filepath.Join(home, ".local", "bin", "android"),
		Log:   filepath.Join(home, "Library", "Logs", "android-mtp.log"),
		Plist: filepath.Join(home, "Library", "LaunchAgents", label+".plist"),
	}
}

func (a agent) plist() []byte {
	var b bytes.Buffer
	if err := plistTmpl.Execute(&b, a); err != nil {
		panic(err)
	}
	return b.Bytes()
}

// install copie le binaire courant hors de ~/Documents, que macOS interdit à launchd de lire,
// puis charge l'agent.
func install(home string) error {
	a := newAgent(home)
	self, err := os.Executable()
	if err != nil {
		return err
	}
	if self, err = filepath.EvalSymlinks(self); err != nil {
		return err
	}

	launchctl("bootout", domain()+"/"+a.Label)
	if self != a.Bin {
		if err := copyExecutable(self, a.Bin); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(a.Plist), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(a.Plist, a.plist(), 0o644); err != nil {
		return err
	}
	if err := launchctl("bootstrap", domain(), a.Plist); err != nil {
		return err
	}

	fmt.Printf("installé : le téléphone se monte tout seul dans ~/Android, journal dans %s\n", a.Log)
	if _, err := os.Stat("/Library/Filesystems/macfuse.fs"); err != nil {
		fmt.Println("macFUSE manque : brew install --cask macfuse, puis autorise le dans Réglages Système")
	}
	return nil
}

func uninstall(home, dir string) error {
	a := newAgent(home)
	launchctl("bootout", domain()+"/"+a.Label)
	if isMounted(dir) {
		forceUnmount(dir)
	}
	signalPtpcamerad(syscall.SIGCONT)
	for _, f := range []string{a.Plist, a.Bin} {
		if err := os.Remove(f); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	fmt.Println("désinstallé")
	return nil
}

// signalAgent demande à l'agent de remonter un téléphone éjecté.
func signalAgent() error {
	return launchctl("kill", "SIGUSR1", domain()+"/"+label)
}

func domain() string {
	return fmt.Sprintf("gui/%d", os.Getuid())
}

func launchctl(args ...string) error {
	if out, err := exec.Command("launchctl", args...).CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl %s: %v %s", args[0], err, bytes.TrimSpace(out))
	}
	return nil
}

// copyExecutable écrit à côté puis renomme, pour remplacer sans risque un binaire en cours d'exécution.
func copyExecutable(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	tmp := dst + ".new"
	if err := os.WriteFile(tmp, data, 0o755); err != nil {
		return err
	}
	return os.Rename(tmp, dst)
}

func xmlEscape(s string) string {
	var b bytes.Buffer
	xml.EscapeText(&b, []byte(s))
	return b.String()
}
