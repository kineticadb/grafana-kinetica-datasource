# Test Re-enabling Analysis - Grafana 12.3.0+ Scope

**Date**: 2026-04-16
**Context**: Now that we've narrowed support to >=12.3.0, can we re-enable some tests?

---

## Failure Pattern Analysis

### Understanding the Original Failures

When testing >=10.4.0 through 13.x, we saw different failure patterns:

| Test | Grafana 10.4.19 | Grafana 13.1.0 | Pattern |
|------|-----------------|----------------|---------|
| `data-testid*="data-source-settings"` | ❌ Not found | ❌ Not found | Only in 11-12? |
| `getByText('Kinetica')` | ❌ Strict mode (2 elements) | ✅ (implied) | Fixed after 10.4 |
| `[data-panelid]` count | ✅ (implied) | ❌ Returns 0 | Changed in 13.x |
| Query editor row | ❌ Not found | ❌ Not found | Never worked |
| Inspector button | ✅ (implied) | ❌ Not found | Changed in 13.x |

---

## Tests Worth Trying

### 1. Plugin Name Visibility (Low Risk) ⚠️

**Test:**
```typescript
test('should show plugin type information', async ({
  readProvisionedDataSource,
  gotoDataSourceConfigPage,
  page,
}) => {
  const datasource = await readProvisionedDataSource({ fileName: 'datasources.yml' });
  await gotoDataSourceConfigPage(datasource.uid);

  // Wait for page load
  await page.waitForLoadState('networkidle');

  // Check for plugin name
  await expect(page.getByText('Kinetica', { exact: false })).toBeVisible({
    timeout: 10000,
  });
});
```

**Why it might work:**
- Only failed in 10.4.19 (strict mode violation with 2 elements)
- No failures reported in 13.1.0
- Might be stable between 12.3-13.x

**Risks:**
- Could still have 0 or 2+ elements in some 12.x or 13.x versions
- Text rendering might vary

**Verdict:** 🟡 **Worth trying** - Low risk, easy to revert

---

### 2. Settings Container (Medium Risk) ⚠️⚠️

**Test:**
```typescript
test('should display datasource settings page', async ({
  readProvisionedDataSource,
  gotoDataSourceConfigPage,
  page,
}) => {
  const datasource = await readProvisionedDataSource({ fileName: 'datasources.yml' });
  await gotoDataSourceConfigPage(datasource.uid);

  await page.waitForLoadState('networkidle');

  // Check for settings container
  const settingsContainer = page.locator('[data-testid*="data-source-settings"]').first();
  await expect(settingsContainer).toBeVisible({ timeout: 10000 });
});
```

**Why it might NOT work:**
- Failed in BOTH 10.4.19 AND 13.1.0
- Suggests data-testid only exists in intermediate versions (11.x? 12.x?)
- If it only exists in 12.0-12.9 but not 13.x, still unreliable

**Risks:**
- High probability of failure in 13.x
- Might pass in 12.3.0 but fail in 13.0+

**Verdict:** 🔴 **Not recommended** - Failed at both ends of spectrum

---

### 3. Panel Count (High Risk) ❌

**Test:**
```typescript
test('should display multiple panels', async ({
  readProvisionedDashboard,
  gotoDashboardPage,
  page,
}) => {
  const dashboard = await readProvisionedDashboard({ fileName: 'kinetica-sample-dashboard.json' });
  await gotoDashboardPage(dashboard);
  await page.waitForLoadState('networkidle');

  const panels = page.locator('[data-panelid]');
  const panelCount = await panels.count();
  expect(panelCount).toBeGreaterThan(0);
});
```

**Why it won't work:**
- Explicitly failed in Grafana 13.1.0 (returned 0)
- DOM structure changed from 12.x → 13.x
- This is a 12-to-13 transition issue, not a 10-to-13 issue

**Risks:**
- Will definitely fail on 13.x
- Our range includes 13.x

**Verdict:** 🔴 **Do not re-enable** - Known to fail in our range

---

### 4. Query Editor Tests (Extremely High Risk) ❌

**All tests using:**
- `panelEditPage.getQueryEditorRow('A')`
- Inspector button
- Add query button

**Why they won't work:**
- Failed in BOTH 10.4.19 AND 13.1.0
- @grafana/plugin-e2e library methods use version-specific selectors
- These are deeply embedded in the library's implementation

**Risks:**
- Even if they work in 12.3, they fail in 13.x
- Library itself is the problem

**Verdict:** 🔴 **Do not re-enable** - Library limitation

---

## Experimental Approach

### Option 1: Conservative (Recommended)

**Try only the safest test:**

Enable 1 test in configEditor.spec.ts:
```typescript
test('should show plugin type information', async ({ ... }) => {
  // Check for "Kinetica" text
});
```

**Total tests:** 7 → 8 tests

**If this passes in CI across 12.3-13.x:** ✅ Keep it
**If it fails:** ❌ Revert and document

### Option 2: Aggressive (Not Recommended)

Enable multiple tests including settings container.

**Risk:** High chance of CI failures in 13.x
**Benefit:** Minimal - one more test isn't worth flaky CI

---

## Recommendation

### Try Plugin Name Test Only

**Change to make:**

```typescript
// In tests/configEditor.spec.ts
// Move this test OUT of the skip block:

test.describe('Config Editor - Provisioned Datasource', () => {
  test('should load provisioned datasource successfully', async ({ ... }) => {
    // Existing test
  });

  // ADD THIS:
  test('should show plugin type information', async ({
    readProvisionedDataSource,
    gotoDataSourceConfigPage,
    page,
  }) => {
    const datasource = await readProvisionedDataSource({ fileName: 'datasources.yml' });
    await gotoDataSourceConfigPage(datasource.uid);

    await page.waitForLoadState('networkidle');

    // Very loose check - just verify "Kinetica" appears somewhere
    await expect(page.getByText('Kinetica', { exact: false })).toBeVisible({
      timeout: 10000,
    });
  });
});
```

**If successful:** 8 stable tests (was 7)
**If failure:** Document that even this fails, keep 7 tests

---

## Why Not More Tests?

### The Core Issue Remains

Even within 12.3-13.x:
1. **UI still evolves** - Grafana 13.x has different DOM from 12.x
2. **Library methods still version-specific** - @grafana/plugin-e2e has same limitations
3. **Panel structure changed** - [data-panelid] proves 12.x→13.x incompatibility

**The problem wasn't just 10.4 being old** - it's that Grafana UI evolves continuously.

### Realistic Expectations

Narrowing to 12.3.0+ helps by:
- ✅ Eliminating ancient version quirks (10.4 weirdness)
- ✅ Reducing version span (2 years instead of 3+)
- ⚠️ But NOT eliminating version differences entirely

**We still have 12.3.0 through 13.1.x+ in our range** - that's still significant evolution.

---

## Testing Strategy

### If We Try the Plugin Name Test

1. **Implement the change** (add 1 test to configEditor.spec.ts)
2. **Commit separately** so it's easy to revert:
   ```bash
   git commit -m "Experiment: Re-enable plugin name visibility test

   Now that plugin requires Grafana >=12.3.0, this test that only
   failed on 10.4.19 might work. Testing empirically in CI.

   If this fails, will revert and document why."
   ```
3. **Push and watch CI** across all Grafana versions
4. **Analyze results:**
   - If passes on all (12.3, 12.4, 12.5, 13.0, 13.1): ✅ Success! Keep it
   - If fails on any: ❌ Revert with explanation

### If Test Fails

Document in tests:
```typescript
test.describe.skip('Config Editor - Additional Checks (Still unreliable even in 12.3+)', () => {
  // Tested 2026-04-16: "Kinetica" text visibility still fails
  // Even between 12.3.0 and 13.x, element count varies
  // Tried after narrowing to >=12.3.0, still unreliable
});
```

---

## Expected Outcome

### Most Likely: No Improvement

**Prediction:** Even the plugin name test might fail because:
- UI rendering varies even in 12.x minor versions
- Text might appear 0, 1, or 2 times depending on context
- Strict mode violations could still occur

**Probability:** 60% chance of failure

### Best Case: +1 Test

**If successful:** 8 tests instead of 7
- Still far from comprehensive
- Still ~95% manual testing
- But shows narrowing scope helped somewhat

**Probability:** 40% chance of success

### Realistic Assessment

**The version narrowing helps, but not dramatically.**

The fundamental issues:
- ✅ Solves: Ancient version incompatibilities (10.4)
- ⚠️ Helps: Reduces version span
- ❌ Doesn't solve: UI evolution between major versions (12→13)
- ❌ Doesn't solve: @grafana/plugin-e2e library limitations

---

## Experimental Result: Test Failed ❌

**Date**: 2026-04-16
**Test Attempted**: Plugin name visibility (`getByText('Kinetica')`)
**Result**: Failed in CI even within Grafana 12.3-13.x range

### What We Learned

**Hypothesis**: Test only failed in 10.4.19, so narrowing to >=12.3.0 might make it stable.

**Reality**: Test still fails even within narrower range.

**Why**: UI rendering varies enough between 12.3, 12.4, 13.0, 13.1, etc. that text visibility tests remain unreliable.

### Key Insight

**Narrowing from >=10.4.0 to >=12.3.0 does NOT significantly enable more E2E tests.**

The problem isn't just ancient versions being weird - it's that **Grafana UI evolves continuously**, even between minor versions.

### Final Verdict

**7 stable tests is the realistic limit for Grafana 12.3.0+**

No additional tests can be reliably re-enabled.

---

## Conclusion

### Worth Trying: 1 Test ✅ (Tried, Failed)

We attempted to re-enable the plugin name visibility test as an experiment.

**Result:** Failed - still unreliable even in narrower range
**Time invested:** Worth it to know empirically
**Learning:** Confirms UI evolution is the core problem, not just old version quirks

### Not Worth: Settings Container or Panel Tests ✅ (Correct)

These explicitly failed in 13.x, so they won't work in our range. **Analysis was correct.**

### The Reality (Confirmed Empirically)

**Narrowing to 12.3.0+ is valuable for:**
- ✅ Honest compatibility claims
- ✅ Reduced CI matrix (fewer versions to test)
- ✅ Modern API usage
- ✅ Eliminating dependency mismatch

**But NOT for:**
- ❌ Dramatically more E2E tests (still 7, not 15 or 20)
- ❌ Comprehensive UI automation (~5% automated, 95% manual)
- ❌ Avoiding manual testing burden

**We're at the "7 stable tests + heavy manual testing" reality.**

That's the pragmatic conclusion after empirical testing across Grafana 12.3-13.x.
