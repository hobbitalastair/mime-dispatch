package lib

func MergeMetadata(fileMeta, xattrMeta map[string][]string) map[string][]string {
	result := make(map[string][]string)

	for k, v := range fileMeta {
		result[k] = v
	}

	for k, v := range xattrMeta {
		result[k] = v
	}

	return result
}

func MergeWithOverride(fileMeta, xattrMeta map[string][]string) map[string][]string {
	result := make(map[string][]string)

	for k, v := range fileMeta {
		result[k] = v
	}

	for k, v := range xattrMeta {
		if _, exists := result[k]; exists {
			result[k] = v
		} else {
			result[k] = v
		}
	}

	return result
}
