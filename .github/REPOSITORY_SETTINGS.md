# Repository Settings Configuration

This document describes the GitHub repository settings required for workflows to function correctly.

## Required Settings

### 1. Allow GitHub Actions to Create Pull Requests

**Issue:** The "Check Debian Cloud Image Updates" workflow fails with:
```
pull request create failed: GraphQL: GitHub Actions is not permitted to create or approve pull requests (createPullRequest)
```

**Solution:** Enable GitHub Actions to create pull requests in repository settings.

**Steps to Fix:**
1. Go to repository **Settings**
2. Navigate to **Actions** → **General**
3. Scroll to **Workflow permissions**
4. Enable **"Allow GitHub Actions to create and approve pull requests"**
5. Click **Save**

**Why This Is Needed:**
The automated Debian cloud image update workflow needs to create pull requests when new Debian images are detected. Without this permission, the workflow can push branches but fails when attempting to create PRs.

**Current Workaround:**
When the workflow fails, manually create the PR using:
```bash
gh pr create \
  --head "auto-update/debian-cloud-YYYYMMDD" \
  --title "Update Debian cloud image to YYYYMMDD" \
  --body "..." \
  --label "automated,dependencies"
```

### 2. Artifact Storage Management

**Status:** ✅ Already configured with retention policies

Artifact retention has been set to:
- **CI builds:** 7 days
- **Release builds:** 14 days

This prevents the repository from hitting GitHub's artifact storage quota.

## Workflow-Specific Permissions

All workflows have been configured with appropriate permissions in their YAML files:

### check-debian-updates.yml
```yaml
permissions:
  contents: write        # Push branches and tags
  pull-requests: write   # Create pull requests
```

### ci.yml
- Default permissions sufficient
- Artifact uploads use `continue-on-error: true` for resilience

### build-images.yml
- Artifacts are required dependencies between jobs
- Retention set to 14 days to manage storage

## Verification

After enabling the "Allow GitHub Actions to create and approve pull requests" setting:

1. **Test manually:**
   ```bash
   gh workflow run check-debian-updates.yml
   ```

2. **Check the workflow run:**
   ```bash
   gh run watch
   ```

3. **Verify PR creation:**
   The workflow should successfully create a PR when a new Debian image is detected.

## Notes

- The repository already has the required labels: `automated` and `dependencies`
- All workflow permissions are configured correctly in YAML files
- The only missing configuration is the repository-level setting to allow Actions to create PRs
