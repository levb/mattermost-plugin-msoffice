// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package fields

type Fields map[string]Value

func Diff(before, after Fields) (changed bool, added, updated, deleted []string) {
	all := Fields{}
	for k := range before {
		all[k] = nil
	}
	for k := range after {
		all[k] = nil
	}
	for k := range all {
		switch {
		case after[k] == nil:
			deleted = append(deleted, k)

		case before[k] == nil:
			added = append(added, k)

		case !before[k].Equals(after[k]):
			updated = append(updated, k)
		}
	}

	return len(added)+len(updated)+len(deleted) != 0,
		added, updated, deleted
}
