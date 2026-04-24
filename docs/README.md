# Grafana Kinetica Datasource - Documentation

This directory contains comprehensive documentation for the Grafana Kinetica datasource plugin, including technical guides, compliance reports, and work logs.

## Table of Contents

### Plugin Documentation

- [Plugin Validation Guide](./PLUGIN_VALIDATION.md) - How to validate the plugin using Grafana's plugin validator
- [Test Environment Setup](./TEST_ENVIRONMENT.md) - Setting up a local development and test environment
- [Publishing Compliance](./PUBLISHING_COMPLIANCE.md) - Compliance status with Grafana publishing best practices

### Development Work Logs

- [Work Logs Directory](./work-logs/) - Documentation of development sessions and changes

## Quick Navigation

### For Contributors

If you're contributing to this plugin, start with:
1. [Test Environment Setup](./TEST_ENVIRONMENT.md) - Get your local environment running
2. [Work Logs Guide](./work-logs/README.md) - Learn how to document your work

### For Reviewers

If you're reviewing this plugin for publication:
1. [Publishing Compliance](./PUBLISHING_COMPLIANCE.md) - Current compliance status
2. [Plugin Validation](./PLUGIN_VALIDATION.md) - Validation results and requirements

### For Maintainers

If you're maintaining this plugin:
1. [Work Logs](./work-logs/) - Historical record of changes and decisions
2. [Plugin Validation](./PLUGIN_VALIDATION.md) - Validation process for releases

## Documentation Standards

All documentation in this directory follows these principles:

- **Markdown Format**: All docs use GitHub-flavored Markdown
- **Keep Updated**: Documentation should be updated when changes occur
- **Be Specific**: Include code examples, commands, and file paths
- **Date Stamped**: Work logs include dates for tracking
- **Linked**: Documents reference each other for easy navigation

## Project Structure

```
docs/
├── README.md                           # This file - documentation index
├── PLUGIN_VALIDATION.md                # Plugin validation guide
├── TEST_ENVIRONMENT.md                 # Test environment setup
├── PUBLISHING_COMPLIANCE.md            # Publishing compliance report
└── work-logs/                          # Development work documentation
    ├── README.md                       # Work logs guide
    ├── TEMPLATE.md                     # Template for new work sessions
    └── YYYY-MM-DD-description.md       # Individual work session logs
```

## Contributing to Documentation

When adding new documentation:

1. Place technical guides in the root `docs/` directory
2. Place work session logs in `docs/work-logs/`
3. Update this README with links to new documents
4. Follow the existing formatting and style conventions
5. Include code examples and practical instructions

## External Documentation

- [Grafana Plugin Development](https://grafana.com/developers/plugin-tools/)
- [Kinetica Documentation](https://docs.kinetica.com)
- [Repository README](../README.md)
- [User Guide](../user_guide.md)
