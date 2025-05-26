# Git GPG Troubleshooting Guide

This guide helps resolve common Git commit and GPG signing issues in the HybridPipe.io repository.

## Common Issues and Solutions

### 1. GPG Signing Failures

#### Issue: "gpg failed to sign the data"
```bash
error: gpg failed to sign the data
fatal: failed to write commit object
```

#### Solutions:

**Quick Fix (Temporary):**
```bash
# Disable GPG signing for this repository
git config --local commit.gpgsign false

# Commit your changes
git commit -m "Your commit message"

# Re-enable GPG signing later if needed
git config --local commit.gpgsign true
```

**Permanent Fix:**
```bash
# Check if GPG is working
gpg --list-secret-keys --keyid-format LONG

# If no keys exist, generate one
gpg --full-generate-key

# Configure Git to use your GPG key
git config --global user.signingkey YOUR_KEY_ID
git config --global commit.gpgsign true

# Test GPG signing
echo "test" | gpg --clearsign
```

### 2. Remote Repository Issues

#### Issue: Missing remote origin
```bash
fatal: 'origin' does not appear to be a git repository
```

#### Solution:
```bash
# Add the correct remote origin
git remote add origin https://github.com/AnandSGit/HybridPipe.io.git

# Or update existing remote
git remote set-url origin https://github.com/AnandSGit/HybridPipe.io.git

# Verify remote configuration
git remote -v
```

#### Issue: Missing remote branch references
```bash
There is no tracking information for the current branch
```

#### Solution:
```bash
# Fetch remote branches
git fetch origin

# Set up tracking for master branch
git branch --set-upstream-to=origin/master master

# Or push and set upstream
git push -u origin master
```

### 3. Authentication Issues

#### Issue: Authentication failed when pushing
```bash
remote: Support for password authentication was removed
```

#### Solutions:

**Option 1: Use Personal Access Token**
```bash
# Generate a token at: https://github.com/settings/tokens
# Use token as password when prompted

# Or configure Git to use token
git remote set-url origin https://YOUR_TOKEN@github.com/AnandSGit/HybridPipe.io.git
```

**Option 2: Use SSH Key**
```bash
# Generate SSH key if you don't have one
ssh-keygen -t ed25519 -C "your_email@example.com"

# Add SSH key to GitHub account
cat ~/.ssh/id_ed25519.pub

# Update remote to use SSH
git remote set-url origin git@github.com:AnandSGit/HybridPipe.io.git
```

### 4. Commit Message Issues

#### Issue: Empty commit message
```bash
Aborting commit due to empty commit message
```

#### Solution:
```bash
# Use -m flag for commit message
git commit -m "Your descriptive commit message"

# Or use multi-line commit message
git commit -m "Short summary" -m "Detailed description"
```

## Automated Fix Scripts

### Quick Fix Script
Run this script to automatically fix common issues:

```bash
# Make the script executable
chmod +x fix_git_issues.sh

# Run the fix script
./fix_git_issues.sh
```

### Phase 3 Commit Script
Use this script to commit the Phase 3 implementation:

```bash
# Make the script executable
chmod +x commit_phase3.sh

# Run the commit script
./commit_phase3.sh
```

## Git Configuration Best Practices

### Recommended Global Configuration
```bash
# Set your identity
git config --global user.name "Your Name"
git config --global user.email "your.email@example.com"

# Set default branch name
git config --global init.defaultBranch master

# Enable helpful features
git config --global color.ui auto
git config --global core.autocrlf input  # For macOS/Linux
git config --global pull.rebase false
```

### Repository-Specific Configuration
```bash
# For HybridPipe.io repository
cd /path/to/HybridPipe.io

# Disable GPG signing if causing issues
git config --local commit.gpgsign false

# Set up remote tracking
git config --local branch.master.remote origin
git config --local branch.master.merge refs/heads/master
```

## Troubleshooting Commands

### Diagnostic Commands
```bash
# Check Git configuration
git config --list

# Check repository status
git status
git log --oneline -5

# Check remote configuration
git remote -v
git branch -r

# Check GPG configuration
gpg --list-secret-keys
git config user.signingkey
```

### Reset Commands (Use with caution)
```bash
# Reset last commit (keep changes)
git reset --soft HEAD~1

# Reset staging area
git reset HEAD

# Discard all local changes
git checkout -- .
```

## Emergency Recovery

### If Repository is in Bad State
```bash
# Create backup branch
git branch backup-$(date +%Y%m%d)

# Reset to last known good state
git fetch origin
git reset --hard origin/master

# Or start fresh (nuclear option)
git clone https://github.com/AnandSGit/HybridPipe.io.git fresh-clone
```

### If Commit History is Corrupted
```bash
# Check repository integrity
git fsck

# Recover from reflog
git reflog
git reset --hard HEAD@{n}  # where n is the good state
```

## Prevention Tips

1. **Regular Commits**: Commit frequently to avoid large, complex commits
2. **Test GPG**: Test GPG signing before important commits
3. **Backup Branches**: Create backup branches before major changes
4. **Remote Sync**: Regularly fetch and push to stay in sync
5. **Configuration Check**: Verify Git configuration before starting work

## Getting Help

If issues persist:

1. Check Git documentation: `git help <command>`
2. Check GitHub documentation: https://docs.github.com/
3. Verify repository permissions on GitHub
4. Contact repository maintainer if needed

## Phase 3 Specific Notes

For the Phase 3 implementation commit:

- All protocol files are in `hybridpipe/protocols/`
- Test files are in `tests/`
- Documentation is in `docs/`
- Temporary files have been removed
- Repository is clean and ready for production

The commit includes:
- ✅ NATS protocol implementation
- ✅ ZeroMQ protocol implementation  
- ✅ AMQP 1.0 protocol implementation
- ✅ Go compatibility layer
- ✅ Comprehensive test suite
- ✅ Updated documentation
- ✅ CI/CD pipeline configuration
