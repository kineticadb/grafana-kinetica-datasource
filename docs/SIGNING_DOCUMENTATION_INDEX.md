# Plugin Signing Documentation - Complete Index

This index helps you find the right documentation for plugin signing based on your needs.

## 📚 Available Guides

### 1. **Quick Reference** - Start Here! ⚡
**File**: [SIGNING_QUICK_REFERENCE.md](SIGNING_QUICK_REFERENCE.md)

**Best for**: Getting up and running in 5 minutes

**Contains**:
- ✅ Minimal steps to enable signing
- ✅ Essential commands only
- ✅ Quick troubleshooting
- ✅ No extra details

**Use when**: You just want to enable signing and move on.

---

### 2. **Step-by-Step Guide** - Complete Walkthrough 📖
**File**: [PLUGIN_SIGNING_STEP_BY_STEP.md](PLUGIN_SIGNING_STEP_BY_STEP.md)

**Best for**: First-time setup or detailed understanding

**Contains**:
- ✅ Detailed instructions with screenshots descriptions
- ✅ Every step explained thoroughly
- ✅ Local testing instructions
- ✅ CI/CD verification steps
- ✅ Comprehensive troubleshooting
- ✅ Security best practices
- ✅ Validation procedures

**Use when**:
- Setting up signing for the first time
- Need to understand what each step does
- Troubleshooting complex issues
- Want to test signing locally before pushing

---

### 3. **Workflow Diagram** - Visual Guide 🎨
**File**: [SIGNING_WORKFLOW_DIAGRAM.md](SIGNING_WORKFLOW_DIAGRAM.md)

**Best for**: Understanding the overall process visually

**Contains**:
- ✅ ASCII art workflow diagrams
- ✅ File structure visualizations
- ✅ Security flow diagrams
- ✅ Decision trees for troubleshooting
- ✅ MANIFEST.txt structure explained

**Use when**:
- Want to see the big picture
- Need to explain signing to team members
- Understanding how components interact
- Debugging workflow issues

---

### 4. **Original Signing Guide** - Reference 📋
**File**: [PLUGIN_SIGNING_GUIDE.md](PLUGIN_SIGNING_GUIDE.md)

**Best for**: Alternative perspective and additional context

**Contains**:
- ✅ Why signing is required
- ✅ When signing is optional
- ✅ CI/CD configuration examples
- ✅ Troubleshooting common errors
- ✅ Additional resources

**Use when**:
- The other guides don't answer your question
- Need more context on why signing exists
- Looking for alternative explanations

---

## 🎯 Choose Your Path

### Path 1: "Just make it work" ⚡
**Time**: 5 minutes

1. Read: [SIGNING_QUICK_REFERENCE.md](SIGNING_QUICK_REFERENCE.md)
2. Follow the 4 steps
3. Done!

---

### Path 2: "I want to understand it" 📚
**Time**: 30 minutes

1. Start: [PLUGIN_SIGNING_STEP_BY_STEP.md](PLUGIN_SIGNING_STEP_BY_STEP.md)
2. Follow along step-by-step
3. Test locally (optional)
4. Reference: [SIGNING_WORKFLOW_DIAGRAM.md](SIGNING_WORKFLOW_DIAGRAM.md) for visuals

---

### Path 3: "I'm presenting to the team" 👥
**Time**: Preparation + presentation

1. Review: [SIGNING_WORKFLOW_DIAGRAM.md](SIGNING_WORKFLOW_DIAGRAM.md)
2. Explain using diagrams
3. Hand out: [SIGNING_QUICK_REFERENCE.md](SIGNING_QUICK_REFERENCE.md)
4. Q&A: [PLUGIN_SIGNING_STEP_BY_STEP.md](PLUGIN_SIGNING_STEP_BY_STEP.md)

---

### Path 4: "Something's broken" 🔧
**Time**: Varies

1. Quick check: [SIGNING_QUICK_REFERENCE.md](SIGNING_QUICK_REFERENCE.md) troubleshooting section
2. If not solved: [PLUGIN_SIGNING_STEP_BY_STEP.md](PLUGIN_SIGNING_STEP_BY_STEP.md) troubleshooting section
3. Understand flow: [SIGNING_WORKFLOW_DIAGRAM.md](SIGNING_WORKFLOW_DIAGRAM.md) decision tree

---

## 📋 Related Documentation

### Grafana Submission
- [GRAFANA_SUBMISSION_FIXES.md](GRAFANA_SUBMISSION_FIXES.md) - Fix validation errors
- [PACKAGE_VERIFICATION.md](PACKAGE_VERIFICATION.md) - Verify package before submission

### Build & Release
- [../README.md](../README.md) - Main project documentation
- [../.github/workflows/ci.yml](../.github/workflows/ci.yml) - CI workflow
- [../.github/workflows/release.yml](../.github/workflows/release.yml) - Release workflow

---

## 🔑 Key Concepts Summary

### What is Plugin Signing?
Plugin signing adds a cryptographic signature to your plugin package that proves:
- The plugin comes from you (authenticity)
- The plugin hasn't been modified (integrity)
- Grafana can trust the plugin source

### How Does It Work?
```
1. Build plugin → Generate files
2. Sign plugin → Calculate hashes → Get Grafana signature
3. Create MANIFEST.txt → Contains hashes + signature
4. Package plugin → Include MANIFEST.txt
5. Upload to Grafana → Grafana validates signature
6. Users install → Grafana verifies unchanged
```

### What You Need
1. **Grafana Cloud account** (free)
2. **Access Policy token** with `plugins:write` scope
3. **GitHub Secret** named `GRAFANA_ACCESS_POLICY_TOKEN`

### What Happens Automatically
Once configured, every push and release:
- ✅ CI builds the plugin
- ✅ CI signs the plugin
- ✅ CI creates signed package
- ✅ Release includes MANIFEST.txt

---

## 🛠️ Quick Commands Reference

### Setup (one-time)
```bash
# 1. Get token from https://grafana.com/orgs/YOUR_ORG/access-policies
# 2. Add to https://github.com/kineticadb/grafana-kinetica-datasource/settings/secrets/actions
```

### Local Signing (optional)
```bash
export GRAFANA_ACCESS_POLICY_TOKEN="glsa_your_token"
npm run build
mage buildAll
npm run sign
ls dist/MANIFEST.txt  # Should exist
```

### Create Release (automatic signing)
```bash
git tag v1.0.0
git push origin v1.0.0
# GitHub Actions signs automatically
```

### Validate Package
```bash
npx @grafana/plugin-validator@latest kinetica-grafana-datasource-1.0.0.zip
```

---

## ❓ Common Questions

### Q: Do I need to sign for development?
**A**: No. Use `GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS` in docker-compose.yaml

### Q: How often do I need to sign?
**A**: CI/CD signs automatically on every build. You don't need to do anything.

### Q: What if the token expires?
**A**: Generate a new token in Grafana Cloud, update GitHub Secret. That's it.

### Q: Can I sign manually?
**A**: Yes, but not recommended. CI/CD signing is more reliable.

### Q: What if signing fails?
**A**: Check the troubleshooting sections in the guides. Most common issue: GitHub Secret not configured.

### Q: Is signing required for Grafana catalog?
**A**: Yes, absolutely required. Unsigned plugins are rejected.

### Q: Does signing cost money?
**A**: No, signing is free with a Grafana Cloud account (also free).

---

## 🚀 Getting Started Checklist

Use this checklist to set up plugin signing:

### Prerequisites
- [ ] Grafana Cloud account created
- [ ] GitHub repository access (admin)
- [ ] Plugin builds successfully locally

### Setup Steps
- [ ] Navigate to Grafana Cloud Access Policies
- [ ] Create new policy with `plugins:write` scope
- [ ] Generate token (save securely!)
- [ ] Add `GRAFANA_ACCESS_POLICY_TOKEN` to GitHub Secrets
- [ ] Verify workflows reference the secret

### Testing
- [ ] Push commit to trigger CI
- [ ] Verify "Sign plugin" step succeeds
- [ ] Download artifact, check MANIFEST.txt exists
- [ ] Create test tag (e.g., v0.0.1-test)
- [ ] Verify release includes signed package

### Validation
- [ ] Run plugin validator on package
- [ ] All checks pass
- [ ] Ready for Grafana submission

---

## 📞 Support

If you encounter issues not covered in the documentation:

1. **Check logs**: GitHub Actions → Failed workflow → Sign plugin step
2. **Verify token**: Grafana Cloud → Access Policies → Check expiration
3. **Verify secret**: GitHub → Settings → Secrets → Check name spelling
4. **Review workflow**: `.github/workflows/ci.yml` and `release.yml`

**Official Resources**:
- [Grafana Plugin Signing Docs](https://grafana.com/developers/plugin-tools/publish-a-plugin/sign-a-plugin)
- [Access Policy Docs](https://grafana.com/docs/grafana-cloud/security-and-account-management/authentication-and-permissions/access-policies/)
- [@grafana/sign-plugin](https://www.npmjs.com/package/@grafana/sign-plugin)

---

## 📝 Document Status

| Document | Status | Last Updated | Purpose |
|----------|--------|--------------|---------|
| SIGNING_QUICK_REFERENCE.md | ✅ Current | 2026-04-20 | Fast setup |
| PLUGIN_SIGNING_STEP_BY_STEP.md | ✅ Current | 2026-04-20 | Complete guide |
| SIGNING_WORKFLOW_DIAGRAM.md | ✅ Current | 2026-04-20 | Visual reference |
| PLUGIN_SIGNING_GUIDE.md | ✅ Current | Previous | Original guide |
| SIGNING_DOCUMENTATION_INDEX.md | ✅ Current | 2026-04-20 | This file |

---

## 🎓 Learning Path Recommendation

**For Developers New to Plugin Signing**:
1. Week 1: Read quick reference, set up signing
2. Week 2: Read step-by-step guide for deeper understanding
3. Week 3: Review workflow diagrams to see big picture
4. Ongoing: Reference troubleshooting as needed

**For DevOps/CI Engineers**:
1. Read workflow diagrams first (understand architecture)
2. Review CI/CD configuration in step-by-step guide
3. Implement using quick reference
4. Set up monitoring for signing failures

**For Project Managers**:
1. Review workflow diagram (high-level overview)
2. Understand timeline: 5 minutes setup + automatic thereafter
3. Know requirement: Must sign for Grafana catalog
4. Budget: Free (Grafana Cloud account is free)

---

**Need to dive in?** Start with [SIGNING_QUICK_REFERENCE.md](SIGNING_QUICK_REFERENCE.md) →
