package git

import (
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v6"
)

// CloneRepo clones a remote Git repository whose URL
// is passed as an argument into a directory also passed as
// an argument. CloneRepo also deletes the .git directory.
func CloneRepo(dirPath string, repoURL string) error {
	_, err := git.PlainClone(dirPath, &git.CloneOptions{
		URL: repoURL,
	})
	if err != nil {
		return err
	}

	if err = os.RemoveAll(filepath.Join(dirPath, ".git")); err != nil {
		return err
	}

	return nil
}
