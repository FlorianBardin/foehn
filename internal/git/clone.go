package git

import (
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v6"
)

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
