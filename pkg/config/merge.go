package config

// DeepMerge recursively merges source into target.
// Returns the merged result. Target is mutated in-place for performance.
//
// Merge rules:
//   - Maps: recursively merge keys (source overwrites target for shared keys)
//   - Slices: concatenate (target + source)
//   - Primitives: source replaces target
//   - Type conflicts: source type takes precedence
//   - nil target treated as empty map
//   - nil source is a no-op (returns target unchanged)
//
// Example:
//
//	target := ConfigObject{"a": map[string]interface{}{"b": 1}}
//	source := ConfigObject{"a": map[string]interface{}{"c": 2}}
//	result := DeepMerge(target, source)
//	// result: {"a": {"b": 1, "c": 2}}
func DeepMerge(target, source ConfigObject) ConfigObject {
	if source == nil {
		if target == nil {
			return make(ConfigObject)
		}
		return target
	}
	if target == nil {
		target = make(ConfigObject)
	}

	for key, sourceValue := range source {
		targetValue, exists := target[key]
		if !exists {
			target[key] = sourceValue
			continue
		}

		// Both values exist - check types
		// Handle both map[string]interface{} and ConfigObject
		var targetMap, sourceMap map[string]interface{}
		targetIsMap := false
		sourceIsMap := false

		switch tv := targetValue.(type) {
		case map[string]interface{}:
			targetMap = tv
			targetIsMap = true
		case ConfigObject:
			targetMap = map[string]interface{}(tv)
			targetIsMap = true
		}

		switch sv := sourceValue.(type) {
		case map[string]interface{}:
			sourceMap = sv
			sourceIsMap = true
		case ConfigObject:
			sourceMap = map[string]interface{}(sv)
			sourceIsMap = true
		}

		if targetIsMap && sourceIsMap {
			// Both are maps - recursively merge
			target[key] = DeepMerge(targetMap, sourceMap)
			continue
		}

		targetSlice, targetIsSlice := targetValue.([]interface{})
		sourceSlice, sourceIsSlice := sourceValue.([]interface{})

		if targetIsSlice && sourceIsSlice {
			// Both are slices - concatenate
			merged := make([]interface{}, len(targetSlice)+len(sourceSlice))
			copy(merged, targetSlice)
			copy(merged[len(targetSlice):], sourceSlice)
			target[key] = merged
			continue
		}

		// Type conflict or primitives - source wins
		target[key] = sourceValue
	}

	return target
}
