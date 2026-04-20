package ui

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pyrorhythm/moonshine/internal/reconciler"
)

// Banner prints the moonshine brand header.
func Banner() {
	fmt.Println(styleBrand.Render("  moonshine ") + styleMuted.Render("declarative package manager"))
	fmt.Println()
}

// Success prints a success message.
func Success(msg string) { fmt.Println(styleSuccess.Render("✓ " + msg)) }

// Warn prints a warning message.
func Warn(msg string) { fmt.Println(styleWarn.Render("⚠ " + msg)) }

// Error prints an error message to stderr.
func Error(msg string) { fmt.Fprintln(os.Stderr, styleError.Render("✗ "+msg)) }

// Info prints an informational message.
func Info(msg string) { fmt.Println(styleMuted.Render("  " + msg)) }

// PrintDiff renders a DiffResult as a coloured diff to w.
func PrintDiff(w io.Writer, result reconciler.DiffResult) {
	if !result.HasChanges() {
		fmt.Fprintln(w, styleSuccess.Render("Nothing to do — system matches moonfile."))
		return
	}
	for _, a := range result.Actions {
		if a.Kind == reconciler.ActionNone {
			continue
		}
		var line string
		backendLabel := styleMuted.Render("  [" + a.BackendName + "]")
		switch a.Kind {
		case reconciler.ActionInstall:
			name := styleName.Render(a.Package.Name())
			ver := ""
			if v := a.Package.Get("version"); v != "" {
				ver = " " + styleVersion.Render("@"+v)
			}
			line = fmt.Sprintf("  %s %s%s%s", styleAdd.Render("+"), name, ver, backendLabel)
		case reconciler.ActionUpgrade:
			name := styleName.Render(a.Package.Name())
			from := styleVersion.Render(a.Current.Version)
			to := styleVersion.Render(a.Package.Get("version"))
			line = fmt.Sprintf("  %s %s %s → %s%s", styleChange.Render("~"), name, from, to, backendLabel)
		case reconciler.ActionUninstall:
			name := styleName.Render(a.Current.Name)
			line = fmt.Sprintf("  %s %s%s", styleRemove.Render("-"), name, backendLabel)
		}
		fmt.Fprintln(w, line)
	}
}

// PrintStatus renders a status summary to w.
func PrintStatus(w io.Writer, result reconciler.DiffResult) {
	installs := result.ByKind(reconciler.ActionInstall)
	upgrades := result.ByKind(reconciler.ActionUpgrade)
	removes := result.ByKind(reconciler.ActionUninstall)
	upToDate := result.ByKind(reconciler.ActionNone)

	if len(installs)+len(upgrades)+len(removes) == 0 {
		fmt.Fprintln(w, styleSuccess.Render("✓ All packages up to date"))
	} else {
		if len(installs) > 0 {
			fmt.Fprintln(w, styleAdd.Render(fmt.Sprintf("  %d to install", len(installs))))
		}
		if len(upgrades) > 0 {
			fmt.Fprintln(w, styleChange.Render(fmt.Sprintf("  %d to upgrade", len(upgrades))))
		}
		if len(removes) > 0 {
			fmt.Fprintln(w, styleRemove.Render(fmt.Sprintf("  %d to remove", len(removes))))
		}
	}
	if len(upToDate) > 0 {
		names := make([]string, 0, len(upToDate))
		for _, a := range upToDate {
			names = append(names, a.Package.Name())
		}
		fmt.Fprintln(w, styleMuted.Render("  up to date: "+strings.Join(names, ", ")))
	}
}
