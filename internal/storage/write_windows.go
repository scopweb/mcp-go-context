//go:build windows

package storage

func replaceFile(src, dst string) error {
	return moveReplace(src, dst)
}
