# Grafana create-plugin Tool and Default Versions

**Date**: 2026-04-16
**Issue**: Understanding where `>=10.4.0` came from in plugin.json

---

## What We Found

### Your Question
> "I think the plugin.json was created by the Grafana create-plugin utility (CLI tool), what can we do about that?"

**Answer:** You're likely correct! And it doesn't matter - **you should still change it**.

---

## Research: What Does create-plugin Set?

### Evidence from Grafana's Official Examples

I researched what `@grafana/create-plugin` and Grafana's templates actually use:

**1. Deprecated `grafana-starter-datasource` (old template):**
```json
"grafanaDependency": ">=7.0.0"
```
Source: [grafana-starter-datasource/plugin.json](https://github.com/grafana/grafana-starter-datasource/blob/master/src/plugin.json)

**Note:** This repository is **deprecated** - not the current recommendation.

**2. Grafana Infinity Datasource (community plugin):**
```json
"grafanaDependency": ">=10.4.8"
```
Source: [grafana-infinity-datasource/plugin.json](https://github.com/grafana/grafana-infinity-datasource/blob/main/src/plugin.json)

**3. MySQL Datasource (built-in):**
```json
"grafanaDependency": ">=10.4.0"
```
Source: [MySQL plugin.json](https://github.com/grafana/grafana/blob/main/public/app/plugins/datasource/mysql/plugin.json)

**4. Recent Plugin Examples (2026):**
Multiple examples show `>=10.0.0` as common

### What This Tells Us

**The `>=10.4.0` in your plugin.json likely came from:**

1. **The create-plugin tool using a template** that included this as a conservative default
2. **Grafana 10.4 being the LTS version** when the tool was run or template was updated
3. **A reasonable starting point** that the tool provides

**But this is just a starting point!** The tool can't know:
- What SDK version you'll actually use
- What APIs you'll depend on
- What features you'll implement
- What your actual compatibility needs are

---

## Important: plugin.json Is NOT Immutable

### What create-plugin Actually Does

The `@grafana/create-plugin` tool:
1. **Scaffolds** a plugin structure
2. **Provides templates** with reasonable defaults
3. **Gets you started** quickly

It does NOT:
- Lock you into specific versions
- Require you to support all versions it suggests
- Know your actual requirements
- Automatically update when you change SDK versions

### The Tool's Philosophy

From [Grafana Plugin Tools Documentation](https://grafana.com/developers/plugin-tools/):

The tool provides:
> "A CLI tool for scaffolding a new plugin"

It's meant to:
- Get you started quickly
- Provide working defaults
- Be modified to your needs

**The generated files are YOUR starting point, not YOUR final configuration.**

---

## What You SHOULD Do

### 1. Treat plugin.json as Configuration, Not Code

Just like you would:
- Change the plugin name from "My Plugin" to "Kinetica"
- Update the description
- Add your own metadata
- Customize the features

**You should also:**
- **Update `grafanaDependency` to match your reality**

### 2. The One-Line Fix Still Applies

```bash
# Edit src/plugin.json
# Change:
"grafanaDependency": ">=10.4.0"

# To:
"grafanaDependency": ">=12.3.0"
```

**This is:**
- ✅ Completely normal
- ✅ Expected customization
- ✅ Best practice
- ✅ What other plugin developers do

### 3. Think of It Like package.json

When you run `npm init`, it creates package.json with defaults:
```json
{
  "version": "1.0.0",
  "description": "",
  "main": "index.js"
}
```

**Do you keep these defaults?** NO! You customize them!

Same with plugin.json - the tool gives you a starting point, **you configure it for your needs**.

---

## Real-World Examples

### Grafana Infinity Datasource
Started with some default, now uses:
```json
"grafanaDependency": ">=10.4.8"
```

**Very specific version!** They chose what works for their plugin.

### MySQL Datasource
Uses:
```json
"grafanaDependency": ">=10.4.0"
```

But it's **built-in** to Grafana, so it's built with matching SDK versions for each release.

### Your Plugin Should Match Your SDK

You use:
```json
"@grafana/data": "^12.3.0"
```

So you should declare:
```json
"grafanaDependency": ">=12.3.0"
```

---

## Why Tool Doesn't Set Higher Versions

### The Tool's Dilemma

If `@grafana/create-plugin` set `grafanaDependency` to the latest version:

**Problem:** Plugin wouldn't work on slightly older Grafana installations
- User on Grafana 11.5 couldn't use plugin declaring `>=12.0.0`
- Creates compatibility issues
- Limits adoption

**Solution:** Tool sets a **conservative default**
- Errs on side of broader compatibility
- Lets YOU decide your actual requirements
- Doesn't force you to support old versions you don't test

---

## Documentation: What Grafana Says

### From [Metadata (plugin.json) Reference](https://grafana.com/developers/plugin-tools/reference/plugin-json):

> **grafanaDependency** (required): "Required Grafana version for this plugin. Validated using node-semver."

**Key word: "Required"**

This means: "What version does MY plugin need?"

Not: "What version did the tool suggest?"

### From [Best Practices](https://grafana.com/developers/plugin-tools/key-concepts/best-practices):

> "build for a version of Grafana later than v10.0"

This is guidance, not a requirement. Many plugins target v11.0+ or v12.0+.

---

## The Mental Model

Think of `create-plugin` like a template:

```
@grafana/create-plugin
       ↓
   [Template Files]
       ↓
   YOUR PLUGIN
       ↓
[Customization Required]
       ↓
  [Production Ready]
```

**Where you are:**
- ✅ Have template files
- ✅ Identified customization needed
- ⏳ Need to apply customization
- ⏳ Production ready after customization

---

## What Other Developers Do

### Standard Practice

1. **Run create-plugin** → Get template with defaults
2. **Develop plugin** → Use SDK features you need
3. **Test plugin** → Discover actual minimum version
4. **Update plugin.json** → Reflect reality
5. **Publish** → Users get accurate requirements

### What They DON'T Do

❌ Keep tool defaults blindly
❌ Assume tool knows your requirements
❌ Test against all versions tool suggests
❌ Build with one SDK but claim to support older versions

---

## Addressing Your Concern

### "But the tool set it..."

**Counterpoint:** The tool also set:
```json
"name": "Grafana Data Source Plugin Template",
"id": "myorgid-simple-datasource",
"description": "",
"author": {
  "name": "Your Name"
}
```

**Did you keep these?** NO! You changed them to:
```json
"name": "Kinetica",
"id": "kinetica-datasource",
"description": "Connect Grafana to Kinetica...",
"author": {
  "name": "Kinetica"
}
```

**Same principle applies to `grafanaDependency`!**

---

## The Professional Approach

### What Professional Plugin Developers Do

1. **Start with template** (create-plugin)
2. **Customize configuration** (all metadata)
3. **Match declarations to reality** (SDK version = min Grafana version)
4. **Test what you claim** (only test versions you support)
5. **Document choices** (README, CHANGELOG)

### What They Avoid

1. ❌ Leaving template defaults
2. ❌ Declaring support they don't test
3. ❌ Mismatching SDK and Grafana versions
4. ❌ Assuming tool knows their requirements

---

## Recommendation: Change It!

### It's Normal and Expected

Changing `grafanaDependency` from the tool default is:
- ✅ **Normal** - All mature plugins do this
- ✅ **Expected** - Tool gives starting point, you configure
- ✅ **Professional** - Shows you understand your dependencies
- ✅ **Honest** - Reflects actual compatibility

### The Change

```diff
  "dependencies": {
-   "grafanaDependency": ">=10.4.0",
+   "grafanaDependency": ">=12.3.0",
    "plugins": []
  }
```

**One line. Five minutes. Problem solved.**

---

## FAQ

### Q: "Won't this break existing users?"

**A:** Users on Grafana 10.4-12.2 can:
- Stay on older plugin version
- Upgrade Grafana (recommended - 10.4 is old)
- Use a different plugin

Most users should be on Grafana 12.x+ by now (released 2024).

### Q: "Should I ask Grafana to fix the tool?"

**A:** No need! The tool is working as designed:
- Provides conservative default
- Lets you customize
- Doesn't force specific versions

### Q: "What if I want to support older versions?"

**A:** Then:
1. Downgrade SDK to `^10.4.0` in package.json
2. Only use APIs available in 10.4
3. Test against all claimed versions
4. Accept minimal E2E automation
5. Heavy manual testing required

**Not recommended** unless business requires it.

### Q: "Is this documented?"

**A:** Not explicitly, because it's assumed knowledge:
- Templates are starting points
- Configuration should match reality
- Professional development requires customization

---

## Analogy: Building a House

**Architect gives you blueprints:**
- Default: 3 bedrooms, 2 bathrooms
- Standard layouts
- Common materials

**Do you build exactly from blueprints?** NO!

You customize:
- Number of rooms (your family size)
- Materials (your budget)
- Layout (your preferences)

**Same with plugin templates:**
- Tool gives starting point
- You customize for YOUR needs
- Final product matches YOUR requirements

---

## Bottom Line

### The Tool Did Its Job

`@grafana/create-plugin` provided:
- ✅ Working structure
- ✅ Reasonable defaults
- ✅ Starting point for customization

### Now You Do Your Job

**Customize it to match reality:**
- SDK version: 12.3.0
- Min Grafana: Should also be 12.3.0
- Update plugin.json accordingly

### This Is Normal

**Every professional plugin developer:**
1. Uses create-plugin
2. Gets template with defaults
3. **Changes defaults to match their needs**
4. Publishes customized plugin

**You're on step 3. That's expected!**

---

## Next Steps

1. **Don't overthink it** - Template values are meant to be changed
2. **Update plugin.json** - One line change to `>=12.3.0`
3. **Document the change** - Note in CHANGELOG
4. **Test** - CI will now test appropriate versions
5. **Ship it** - Users get accurate requirements

**The tool gave you a starting point. You make it production-ready.**

That's how it's supposed to work! 🎯
