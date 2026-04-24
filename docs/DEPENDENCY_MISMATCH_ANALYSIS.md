# Critical Dependency Mismatch Analysis

**Date**: 2026-04-16
**Issue**: CI tests against Grafana 10.4+ but plugin is built with SDK 12.3.0

---

## The Problem

There's a **critical mismatch** between what the plugin claims to support and what it's actually built with:

### Plugin Declaration (src/plugin.json:46)

```json
"dependencies": {
  "grafanaDependency": ">=10.4.0",
  "plugins": []
}
```

**This declares:** "My plugin supports all Grafana versions from 10.4.0 onwards"

### Build Dependencies (package.json:70-74)

```json
"dependencies": {
  "@grafana/data": "^12.3.0",
  "@grafana/i18n": "^12.3.0",
  "@grafana/runtime": "^12.3.0",
  "@grafana/ui": "^12.3.0",
  "@grafana/schema": "^12.3.0",
  ...
}
```

**This means:** "I'm building the plugin using Grafana SDK version 12.3.0"

### CI Testing (.github/workflows/ci.yml:143)

```yaml
- name: Resolve Grafana E2E versions
  id: resolve-versions
  uses: grafana/plugin-actions/e2e-version@e2e-version/v1.1.2
```

**This action:** Reads `plugin.json`'s `grafanaDependency` and tests against **ALL declared versions**

**Result:** CI tests against Grafana 10.4.19, 11.x, 12.x, 13.1.0, etc.

---

## Why This Is a Problem

### 1. **API Compatibility Issues**

Building with SDK 12.3.0 while claiming to support 10.4.0 is problematic:

**Scenario A - Using New APIs:**
- SDK 12.3.0 might expose APIs introduced in Grafana 12.x
- Developer uses these APIs during development
- Plugin breaks when running on Grafana 10.4.x
- Runtime error: "undefined is not a function"

**Scenario B - Deprecated APIs:**
- Grafana 10.4 APIs might be deprecated/removed in 12.x
- SDK 12.3.0 won't have old API signatures
- Can't use patterns that work in 10.4

### 2. **Type Definitions Mismatch**

TypeScript types from SDK 12.3.0 might not match runtime in Grafana 10.4:

```typescript
// This might compile fine with SDK 12.3.0 types
import { SomeInterface } from '@grafana/data';

// But fails at runtime in Grafana 10.4 because:
// - SomeInterface doesn't exist yet in 10.4
// - Or has different shape/methods
```

### 3. **Build Output Incompatibility**

The webpack bundle created using SDK 12.3.0:
- Uses ES6+ features that SDK provides
- Assumes certain Grafana globals exist
- May use module resolution that changed

When loaded into Grafana 10.4:
- Missing global variables
- Incompatible React versions
- Module loading failures

---

## Evidence from CI Failures

This mismatch explains many of our E2E test failures:

### Version-Specific Failures

**Grafana 10.4.19:**
- data-testid selectors don't exist (SDK 12.3.0 expects them)
- UI structure completely different

**Grafana 13.1.0:**
- [data-panelid] returns 0 elements (DOM structure changed)
- Query editor aria-labels don't match

The plugin is built for **one version** (12.3.0) but tested against **many versions** (10.4+).

---

## What Other Plugins Do

### Grafana's Built-in Plugins

Built-in plugins (PostgreSQL, MySQL, Prometheus) are **version-matched:**
- Each Grafana release bundles plugins built with **matching SDK version**
- Grafana 10.4 ships with plugins built with SDK 10.4
- Grafana 12.3 ships with plugins built with SDK 12.3

**They don't claim backward compatibility** - each release is a fresh build.

### External Plugins - Two Approaches

**Approach 1: Narrow Version Support**
```json
"grafanaDependency": ">=12.0.0"
```
- Only support recent versions
- Build with latest SDK
- More features, easier development
- Example: Many modern plugins

**Approach 2: Conservative Building**
```json
"grafanaDependency": ">=10.4.0"
```
- Build with **oldest supported SDK** (10.4.x)
- Use only APIs available in 10.4
- Test against all claimed versions
- Example: Plugins needing wide compatibility

---

## What Should We Do?

### Option 1: Narrow the Support (RECOMMENDED)

**Change plugin.json to:**
```json
"dependencies": {
  "grafanaDependency": ">=12.3.0"
}
```

**Pros:**
- Matches build SDK version
- Can use modern Grafana APIs
- Reduces E2E testing complexity
- More reliable plugin behavior
- Focus on well-tested versions

**Cons:**
- Drops support for Grafana 10.4, 11.x
- Users on older versions can't use the plugin
- Smaller addressable market

**E2E Impact:**
- Still need to test 12.3+, 13.x
- But fewer version variations
- Might enable more stable tests

### Option 2: Build with Oldest SDK

**Change package.json to:**
```json
"dependencies": {
  "@grafana/data": "^10.4.0",
  "@grafana/i18n": "^10.4.0",
  "@grafana/runtime": "^10.4.0",
  "@grafana/ui": "^10.4.0",
  "@grafana/schema": "^10.4.0"
}
```

**Pros:**
- Truly supports claimed versions
- Can only use APIs available in 10.4
- Wide market reach

**Cons:**
- Can't use modern Grafana features
- Development with old SDK
- Limited to 10.4 API surface
- TypeScript types are outdated
- Still need extensive E2E testing

### Option 3: Multi-Version Builds (COMPLEX)

Build **separate plugin versions** for different Grafana versions:
- kinetica-datasource-10.x.zip (built with SDK 10.4)
- kinetica-datasource-11.x.zip (built with SDK 11.0)
- kinetica-datasource-12.x.zip (built with SDK 12.3)
- kinetica-datasource-13.x.zip (built with SDK 13.0)

**Pros:**
- Each version optimized for target Grafana
- Can use version-specific features

**Cons:**
- **EXTREMELY COMPLEX**
- Multiple CI/CD pipelines
- Multiple test matrices
- Multiple releases to manage
- High maintenance burden

---

## Recommendation

### **Choose Option 1: Narrow to >=12.3.0**

**Rationale:**

1. **Current Reality:**
   - We're already building with SDK 12.3.0
   - Plugin behavior is optimized for this version
   - Testing shows issues with older versions

2. **Industry Standard:**
   - Most modern plugins support recent versions only
   - Grafana LTS policy: Focus on supported versions
   - Users on old Grafana should upgrade

3. **E2E Testing:**
   - Reduces test matrix complexity
   - Still tests 12.3, 12.4, 13.0, 13.1, etc.
   - More manageable scope

4. **Development Velocity:**
   - Can use modern Grafana APIs
   - Better TypeScript support
   - Cleaner codebase

5. **User Impact:**
   - Grafana 12.3.0 released: ~2024
   - Most users should be on 12.x or newer
   - Old users can stay on old plugin versions

---

## Implementation Plan

### Step 1: Update plugin.json

```bash
# Edit src/plugin.json
sed -i 's/"grafanaDependency": ">=10.4.0"/"grafanaDependency": ">=12.3.0"/' src/plugin.json
```

### Step 2: Document the Change

Add to CHANGELOG.md:
```markdown
## [Version X.Y.Z] - 2026-04-16

### BREAKING CHANGE
- **Minimum Grafana version increased to 12.3.0**
- Reason: Alignment with build SDK version for reliability
- Users on Grafana 10.4-12.2 should use plugin version X.Y.Z-1 or upgrade Grafana
```

### Step 3: Update Documentation

Update README.md:
```markdown
## Requirements

- Grafana >= 12.3.0
```

### Step 4: Test E2E

After change, CI will test against:
- Grafana 12.3.x
- Grafana 12.4.x
- Grafana 13.0.x
- Grafana 13.1.x

Fewer versions = potentially more stable tests.

### Step 5: Consider Re-enabling Tests

With narrower version range, we might be able to:
- Re-enable some E2E tests that now work consistently
- Use more SDK features reliably
- Reduce skip blocks

---

## Alternative: If Wide Support Is Required

If business requirements demand supporting Grafana 10.4+:

### Must Do:

1. **Downgrade SDK to 10.4.x:**
   ```json
   "@grafana/data": "^10.4.0"
   ```

2. **Avoid Modern APIs:**
   - Only use APIs documented in Grafana 10.4 docs
   - Test locally against Grafana 10.4

3. **Accept E2E Limitations:**
   - Keep current minimal test coverage (7 tests)
   - Heavy manual testing burden
   - Version-specific bugs expected

4. **Version-Specific Code:**
   ```typescript
   // Runtime version detection
   if (config.buildInfo.version.startsWith('10.')) {
     // Use old API
   } else {
     // Use new API
   }
   ```

**This is NOT recommended** due to complexity and maintenance burden.

---

## Conclusion

**The current state is unsustainable:** Building with SDK 12.3.0 while claiming to support 10.4+ creates compatibility issues that manifest as E2E test failures.

**Recommended action:** Update `plugin.json` to `"grafanaDependency": ">=12.3.0"` to match the build reality.

This is a **one-line change** that:
- Fixes the mismatch
- Simplifies testing
- Improves reliability
- Aligns with industry practices
- Enables modern development

**Next step:** Decide on minimum supported version and update accordingly.
