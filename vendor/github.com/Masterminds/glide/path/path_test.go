package path

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

const testdata = "../testdata/path"

func TestGlideWD(t *testing.T) {
	wd := filepath.Join(testdata, "a/b/c")
	found, err := GlideWD(wd)
	if err != nil {
		t.Errorf("Failed to get Glide directory: %s", err)
	}

	if found != filepath.Join(testdata, "a") {
		t.Errorf("Expected %s to match %s", found, filepath.Join(wd, "a"))
	}

	// This should fail
	wd = "/No/Such/Dir"
	found, err = GlideWD(wd)
	if err == nil {
		t.Errorf("Expected to get an error on a non-existent directory, not %s", found)
	}

}

func TestVendor(t *testing.T) {
	td, err := filepath.Abs(testdata)
	if err != nil {
		t.Fatal(err)
	}
	wd, _ := os.Getwd()

	os.Chdir(filepath.Join(td, "a", "b", "c"))
	res, err := Vendor()
	if err != nil {
		t.Errorf("Failed to resolve vendor directory: %s", err)
	}
	expect := filepath.Join(td, "a", "vendor")
	if res != expect {
		t.Errorf("Failed to find vendor: expected %s got %s", expect, res)
	}

	os.Chdir(filepath.Join(td, "x", "y", "z"))
	res, err = Vendor()
	if err != nil {
		t.Errorf("Failed to resolve vendor directory: %s", err)
	}

	// Windows symlinks are different than *nix and they can be inconsistent.
	// The current testing only works for *nix testing and windows doesn't follow
	// the symlinks. If this is a vendor.lnk file in windows this won't work for
	// the go toolchain. If this is a windows link you need access to create one
	// which isn't consistent.
	// If there is a better way would love to know.
	if runtime.GOOS == "windows" {
		expect = filepath.Join(td, "x", "vendor")
	} else {
		expect = filepath.Join(td, "x", "symlinked_vendor")
	}
	if res != expect {
		t.Errorf("Failed to find vendor: expected %s got %s", expect, res)
	}

	os.Chdir(wd)
}
func TestGlide(t *testing.T) {
	wd, _ := os.Getwd()
	td, err := filepath.Abs(testdata)
	if err != nil {
		t.Fatal(err)
	}
	os.Chdir(filepath.Join(td, "a/b/c"))
	res, err := Glide()
	if err != nil {
		t.Errorf("Failed to resolve vendor directory: %s", err)
	}
	expect := filepath.Join(td, "a", "glide.yaml")
	if res != expect {
		t.Errorf("Failed to find vendor: expected %s got %s", expect, res)
	}
	os.Chdir(wd)
}

func TestCustomRemoveAll(t *testing.T) {
	td, err := filepath.Abs(testdata)
	if err != nil {
		t.Fatal(err)
	}
	// test that deleting a non-existent directory does not throw an error
	err = CustomRemoveAll(filepath.Join(td, "directory/doesnt/exist"))
	if err != nil {
		t.Errorf("Failed when removing non-existent directory %s", err)
	}
	// test that deleting a path with spaces does not throw an error
	spaceyPath := filepath.Join(td, "10942384 12341234 12343214 324134132323")
	err = os.MkdirAll(spaceyPath, 0777)
	if err != nil {
		t.Fatalf("Failed to make test directory %s", err)
	}
	err = CustomRemoveAll(spaceyPath)
	if err != nil {
		t.Errorf("Errored incorrectly when deleting a path with spaces %s", err)
	}
	if _, err = os.Stat(spaceyPath); !os.IsNotExist(err) {
		t.Errorf("Failed to successfully delete a path with spaces")
	}
}
