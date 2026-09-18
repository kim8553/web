package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSceneCreatorManifestResolvesExistingFilenameCase(t *testing.T) {
	dir := t.TempDir()
	manifest := "[file]\n0=CommonNpc.xml\n1=FuncNpc.xml\n2=AttackNpc_Creator.xml\n3=EventTrigger.xml\n"
	if err := os.WriteFile(filepath.Join(dir, "file.ini"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"commonnpc.xml", "funcnpc.xml", "eventtrigger.xml"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("<object/>"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	paths, err := sceneCreatorPaths(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 || filepath.Base(paths[0]) != "commonnpc.xml" || filepath.Base(paths[1]) != "funcnpc.xml" {
		t.Fatalf("creator paths=%q; want the two actual existing filenames", paths)
	}
}

func TestSceneCreatorManifestRejectsAmbiguousFilenameCase(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows cannot represent case-distinct entries reliably")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "file.ini"), []byte("[file]\n0=CommonNpc.xml\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"commonnpc.xml", "COMMONNPC.xml"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("<object/>"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	_, err := sceneCreatorPaths(dir)
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("case-ambiguous creator should be rejected, got %v", err)
	}
}
