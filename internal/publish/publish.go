package publish

import "context"

// Publish always writes a local markdown file and optionally pushes to Apple Notes.
func Publish(ctx context.Context, notesDir, memoBin, folder string, memoEnabled bool, n Note) (notePath string, err error) {
	notePath, err = WriteMarkdown(notesDir, n)
	if err != nil {
		return "", err
	}

	if !memoEnabled {
		return notePath, nil
	}

	if err := PushMemo(ctx, memoBin, folder, n); err != nil {
		return notePath, err
	}
	return notePath, nil
}
