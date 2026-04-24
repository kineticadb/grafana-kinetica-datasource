# If the Experimental Test Fails

**Test**: Plugin name visibility in configEditor.spec.ts
**Status**: Re-enabled experimentally after narrowing to Grafana >=12.3.0

---

## What We're Testing

After narrowing plugin support from `>=10.4.0` to `>=12.3.0`, we re-enabled ONE test that only failed in the old version range:

```typescript
test('should show plugin type information', async ({ ... }) => {
  await expect(page.getByText('Kinetica', { exact: false })).toBeVisible({
    timeout: 10000,
  });
});
```

**Hypothesis**: This test failed in 10.4.19 but might work between 12.3-13.x

---

## If This Test Fails in CI

### Step 1: Check Which Version Failed

Look at the CI logs to see which Grafana version(s) caused the failure:
- Grafana 12.3.x - Unexpected! Should work
- Grafana 12.4.x - Possible, UI might vary
- Grafana 13.x - More likely, UI evolved from 12.x

### Step 2: Check the Error

**Possible errors:**

**A) Strict Mode Violation**
```
Error: strict mode violation: getByText('Kinetica') resolved to 2 elements
```
**Meaning**: "Kinetica" text appears multiple times on page

**B) Not Found**
```
Error: element not found: getByText('Kinetica')
```
**Meaning**: Text doesn't appear at all

**C) Timeout**
```
Error: Timeout waiting for element
```
**Meaning**: Page structure different, text not rendering

### Step 3: Revert the Test

If the test fails on ANY Grafana version in our range, revert it:

```bash
# Edit tests/configEditor.spec.ts
# Move the test back into the skip block
```

**Specific changes:**

1. Remove the test from the active describe block
2. Add it to the skip block with explanation
3. Update documentation

### Step 4: Update Files

**tests/configEditor.spec.ts:**
```typescript
test.describe.skip('Config Editor - Detailed Checks (Still unreliable even in 12.3+)', () => {
  // TESTED 2026-04-16: Plugin name visibility still unreliable
  // Re-enabled after narrowing to >=12.3.0, but still fails in CI
  // Error: [describe the error from logs]
  // Versions tested: 12.3.x, 12.4.x, 13.x
  // Conclusion: Even with narrower range, text rendering varies
  //
  // test('should show plugin type information', async ({ ... }) => {
  //   await expect(page.getByText('Kinetica', { exact: false })).toBeVisible();
  // });
});
```

**E2E_TESTS_README.md:**
```markdown
**Test Count**:
- **7 passing tests** (stable across all versions)
- **28+ skipped tests** (documented with reasons)

**What's Tested**:
- ✅ Alert rules (5 tests)
- ✅ Config editor page loads (1 test)
- ✅ Dashboard loads (1 test)

**Note**: Attempted to re-enable plugin name visibility test after narrowing
to >=12.3.0, but it still failed. Even within 12.3-13.x, text rendering varies.
```

**docs/E2E_TEST_FIX_SUMMARY.md:**
```markdown
### Final Test Count

| Test File | Passing Tests | Status |
|-----------|--------------|--------|
| alertQueries.spec.ts | 5 | Stable |
| configEditor.spec.ts | 1 | Stable |
| dataQueries.spec.ts | 1 | Stable |
| queryEditor.spec.ts | 0 | All skipped |
| **Total** | **7** | **All stable** |

**Experimental Test Failed**: Plugin name visibility test re-enabled after
narrowing to >=12.3.0, but CI showed it still fails. Reverted to 7 stable tests.
```

**docs/TEST_RE_ENABLING_ANALYSIS.md:**

Add section at the end:
```markdown
## Update: Test Failed (if applicable)

**Date**: [Date of CI failure]

The experimental plugin name visibility test failed in CI despite narrowing to >=12.3.0.

**Failure details:**
- Grafana version: [e.g., 13.1.0]
- Error: [Copy exact error from CI]
- Root cause: [e.g., Text appears 2 times, strict mode violation]

**Conclusion:** Even with narrower version range (12.3-13.x), the test is
unreliable. UI rendering varies enough that we can't depend on text visibility.

**Final verdict:** 7 stable tests is the realistic limit for Grafana 12.3.0+
```

### Step 5: Commit the Reversion

```bash
git add tests/configEditor.spec.ts E2E_TESTS_README.md docs/
git commit -m "Revert: Remove experimental plugin name visibility test

The test failed in CI on Grafana [version] with error:
[exact error message]

Attempted to re-enable after narrowing to >=12.3.0, but text rendering
still varies enough to cause failures. Reverting to 7 stable tests.

See docs/IF_EXPERIMENTAL_TEST_FAILS.md for details."
```

---

## If the Test Passes! 🎉

### Congratulations!

The experimental test passed across all Grafana versions (12.3-13.x).

### Update Documentation

**E2E_TESTS_README.md:**
```markdown
**Test Count**:
- **8 passing tests** (all stable)
- **27 skipped tests** (documented with reasons)

**What's Tested**:
- ✅ Alert rules (5 tests)
- ✅ Config editor (2 tests) - Page loads + plugin name visibility
- ✅ Dashboard loads (1 test)

**Success**: Narrowing to >=12.3.0 enabled re-enabling the plugin name
visibility test. It now passes reliably across 12.3-13.x.
```

**tests/configEditor.spec.ts:**

Remove "EXPERIMENT" comment, mark as stable:
```typescript
test('should show plugin type information', async ({ ... }) => {
  // Verifies plugin name appears on config page
  // Stable across Grafana 12.3.0+
  const datasource = await readProvisionedDataSource({ fileName: 'datasources.yml' });
  await gotoDataSourceConfigPage(datasource.uid);
  await page.waitForLoadState('networkidle');
  await expect(page.getByText('Kinetica', { exact: false })).toBeVisible({
    timeout: 10000,
  });
});
```

**docs/TEST_RE_ENABLING_ANALYSIS.md:**

Add success section:
```markdown
## Update: Test Succeeded! ✅

**Date**: [Date of CI success]

The experimental plugin name visibility test PASSED in CI across all Grafana versions!

**Results:**
- Grafana 12.3.x: ✅ Pass
- Grafana 12.4.x: ✅ Pass
- Grafana 13.0.x: ✅ Pass
- Grafana 13.1.x: ✅ Pass

**Conclusion:** Narrowing to >=12.3.0 successfully enabled re-enabling this test.
Text rendering is consistent enough in the 12.3-13.x range.

**Final verdict:** 8 stable tests for Grafana 12.3.0+

This shows that narrowing the version range CAN help, even if only slightly.
```

### Commit the Success

```bash
git add docs/TEST_RE_ENABLING_ANALYSIS.md E2E_TESTS_README.md tests/configEditor.spec.ts
git commit -m "Confirm: Plugin name visibility test stable in 12.3.0+

After narrowing plugin support to >=12.3.0, experimental test passed
across all Grafana versions in CI. Marking as stable.

Test count: 7 → 8 stable tests

This demonstrates that narrowing version range can help test stability,
even if improvement is incremental."
```

---

## Why We Can't Do More

Even if this ONE test succeeds, we learned from analysis that:

### Tests That Will Still Fail

1. **Settings container** (`data-testid*="data-source-settings"`)
   - Failed in BOTH 10.4.19 AND 13.1.0
   - Won't work even with narrower range

2. **Panel counting** (`[data-panelid]`)
   - Explicitly fails in Grafana 13.x
   - This is a 12.x → 13.x change

3. **Query editor tests** (all)
   - @grafana/plugin-e2e library limitation
   - Won't work regardless of range

### The Reality

**Best case:** 8 tests (was 7)
**Realistic:** Probably still 7 tests

**The core issue remains:** UI evolves even within 12.3-13.x range.

---

## Next Steps After Resolution

### If Test Failed
1. Revert (instructions above)
2. Document why it failed
3. Accept 7 stable tests
4. Focus on manual testing quality

### If Test Succeeded
1. Update docs to mark as stable
2. Consider if ANY other tests worth trying
3. But remain conservative - don't get greedy
4. Most tests will still fail

---

## Decision Tree

```
Test fails in CI?
├─ Yes → Revert (follow Step 3-5 above)
│   └─ Accept 7 stable tests
│       └─ Focus on manual testing
│
└─ No (Test passes!) → Great!
    ├─ Update docs
    ├─ Mark as stable
    └─ Consider if 1-2 more tests worth trying
        ├─ Yes → Try ONE MORE, cautiously
        └─ No → Stop at 8, don't push luck
```

---

## The Big Picture

This experiment shows:

✅ **Narrowing version range CAN help**
- Eliminates ancient version quirks
- Reduces span of UI evolution

❌ **But doesn't solve everything**
- UI still evolves within range
- Library limitations remain
- Most tests still won't work

🎯 **The value isn't in test count**
- 8 tests isn't dramatically better than 7
- The value is in: honest declarations, simpler CI, modern APIs

---

## Conclusion

Whether this test passes or fails, **we learned something valuable:**

**If it fails:** Version narrowing helps but not enough for comprehensive testing
**If it passes:** Version narrowing enables SOME improvement, incrementally

Either way: **Manual testing remains the primary quality assurance method.**

That's the pragmatic reality of E2E testing Grafana plugins. ✅
