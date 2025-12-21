# Homebrew Fix Implementation Checklist

## Phase 1: Understanding (5-10 minutes)

- [ ] Read `HOMEBREW_FIX_SUMMARY.md` - Quick overview
- [ ] Read `VISUAL_EXPLANATION.md` - Understand the flow
- [ ] Understand the three problems:
  - [ ] `|| true` suppresses errors silently
  - [ ] 5-minute wait is too aggressive for asset uploads
  - [ ] Fixed retry delay doesn't adapt to timing

## Phase 2: Choose Implementation Method (2 minutes)

Choose ONE approach:

- [ ] **Method A (Recommended):** Use pre-made fixed script
  - Copy `update-homebrew-formula-FIXED.sh` to `scripts/update-homebrew-formula.sh`
  - Update `.github/workflows/release.yml` line 715
  - Test and commit

- [ ] **Method B:** Manual edits with copy-paste
  - Follow `APPLY_FIXES.md` exact code sections
  - Make 3 edits (2 in script, 1 in workflow)
  - Verify changes
  - Test and commit

- [ ] **Method C:** Step-by-step understanding
  - Read `HOMEBREW_FIXES.md` detailed explanation
  - Make changes while understanding why
  - Test as you go

## Phase 3: Backup & Safety (2 minutes)

- [ ] Create backup of current script:
  ```bash
  cp scripts/update-homebrew-formula.sh scripts/update-homebrew-formula.sh.backup
  ```

- [ ] Create new git branch for safety:
  ```bash
  git checkout -b fix/homebrew-update-reliability
  ```

- [ ] Verify git status is clean:
  ```bash
  git status  # Should show backup file as untracked (OK)
  ```

## Phase 4: Apply Fixes (5-15 minutes depending on method)

### If using Method A (Easiest):

```bash
# [ ] Backup current version
cp scripts/update-homebrew-formula.sh scripts/update-homebrew-formula.sh.backup

# [ ] Copy the fixed version
cp update-homebrew-formula-FIXED.sh scripts/update-homebrew-formula.sh

# [ ] Verify it's executable
chmod +x scripts/update-homebrew-formula.sh

# [ ] Edit .github/workflows/release.yml
#     Find line 715: sleep 300
#     Change to: sleep 900

# [ ] Verify the change
grep -A5 'Wait for GitHub release to stabilize' .github/workflows/release.yml | grep sleep
# Should show: sleep 900
```

### If using Method B (Exact copy-paste):

```bash
# [ ] Edit 1: lines 380-382 in scripts/update-homebrew-formula.sh
#     Remove: release_data=$(get_latest_release) || true
#     Add: if release_data=$(get_latest_release); then ... else release_data="" fi

# [ ] Edit 2: lines 376-428 in scripts/update-homebrew-formula.sh
#     Replace entire retry block with exponential backoff version

# [ ] Edit 3: line 715 in .github/workflows/release.yml
#     Change: sleep 300
#     To: sleep 900
```

### If using Method C (Understanding):

See `HOMEBREW_FIXES.md` for detailed step-by-step with explanations.

## Phase 5: Verify Changes (5 minutes)

Run these checks to ensure edits are correct:

```bash
# [ ] Check script syntax is valid
bash -n scripts/update-homebrew-formula.sh
# (Should output nothing if valid)

# [ ] Verify error suppression is removed
! grep -n 'get_latest_release.*|| true' scripts/update-homebrew-formula.sh
# (Should find nothing - exit code 1 is OK, means pattern not found)

# [ ] Verify exponential backoff is present
grep -n 'backoff_delay.*2 \*\*' scripts/update-homebrew-formula.sh
# (Should find: backoff_delay=$((30 * (2 ** ...)))

# [ ] Verify workflow wait time increased
grep -A1 'sleep 9' .github/workflows/release.yml | head -2
# (Should show: sleep 900)

# [ ] Verify old short sleep is gone
! grep -B5 'Update Homebrew Formula' .github/workflows/release.yml | grep 'sleep 300'
# (Should find nothing)
```

Expected output:
```
✓ Script syntax valid (no output from bash -n)
✓ Error suppression removed (grep returns 1, no matches)
✓ Exponential backoff present (grep finds the line)
✓ Wait time increased (shows sleep 900)
✓ Old timeout removed (grep returns 1, no matches)
```

## Phase 6: Stage Changes for Commit

```bash
# [ ] Check what changed
git diff scripts/update-homebrew-formula.sh

# [ ] Check what changed in workflow
git diff .github/workflows/release.yml

# [ ] Stage the changes
git add scripts/update-homebrew-formula.sh .github/workflows/release.yml

# [ ] Verify staging
git status
# Should show: 2 files changed, not staged for commit
# (if using method A, only these 2 should be shown)
```

## Phase 7: Commit Changes

```bash
# [ ] Create commit with clear message
git commit -m "fix: improve Homebrew formula update reliability

Three critical fixes:

1. Remove silent error suppression (|| true)
   - API failures were hidden, causing empty response parsing errors
   - Now errors bubble up and are logged clearly

2. Increase wait time from 5 to 15 minutes
   - Build jobs + uploads + CDN propagation takes 8-15 min
   - 5 minutes was too aggressive, catching assets mid-upload

3. Add exponential backoff retry (30s, 60s, 120s, 240s)
   - Fixed 30s delay wasn't enough for slow uploads
   - Exponential backoff adapts to actual timing

Fixes: Homebrew formula update failing with 'Release not found'
even though release and assets exist"

# [ ] Verify commit
git log -1 --oneline
# Should show: fix: improve Homebrew formula update reliability
```

## Phase 8: Test the Fix

### Test Setup

```bash
# [ ] Stash any uncommitted work (if needed)
#     (You already committed, so skip this)

# [ ] Create a test release tag
git tag v0.7.15-test-$(date +%s)
# (This creates a unique tag like: v0.7.15-test-1640123456)

# [ ] Push the tag to trigger workflow
git push origin --tags

# [ ] Verify push succeeded
git push --tags -n  # Dry run to verify
```

### Monitor Workflow

```bash
# [ ] Go to GitHub Actions in browser:
#     https://github.com/jbeck018/howlerops/actions

# [ ] Find the Release workflow run for your test tag

# [ ] Click on "Update Homebrew Formula" job

# [ ] GOOD SIGNS (look for these):
#     ✓ "Found release with XX assets"
#     ✓ "Successfully fetched release via gh CLI"
#     ✓ "Homebrew formula update completed successfully!"

# [ ] BAD SIGNS (if you see these, something's wrong):
#     ✗ "Release not found or API returned invalid response"
#     ✗ "release_data is EMPTY"
#     ✗ Multiple retry attempts with same error
```

### Verify Success

If Homebrew job succeeds:
```bash
# [ ] Check that the formula was updated in the tap repo:
#     https://github.com/jbeck018/homebrew-howlerops

# [ ] Look for commit message like:
#     "chore: update howlerops to v0.7.15-test-XXXXX"

# [ ] Verify the file content shows correct version and SHA256
```

### If Test Fails

Before rolling back, debug:

```bash
# [ ] Check the actual gh CLI error
gh api repos/jbeck018/howlerops/releases/latest

# [ ] Check if release was created
gh release list | head -1

# [ ] Check assets for your test tag
gh release view v0.7.15-test-XXXXX --json assets

# [ ] Check API rate limit
gh api rate_limit | jq '.rate.remaining'

# [ ] If everything failed, check GitHub token
gh auth status
```

If fix still doesn't work, rollback:
```bash
# [ ] Restore from backup
cp scripts/update-homebrew-formula.sh.backup scripts/update-homebrew-formula.sh

# [ ] Reset workflow file
git checkout .github/workflows/release.yml

# [ ] Check status
git status

# [ ] Read error logs and HOMEBREW_ISSUE_ANALYSIS.md for troubleshooting
```

## Phase 9: Clean Up Test Release (Optional)

```bash
# [ ] Delete the test tag locally
git tag -d v0.7.15-test-XXXXX

# [ ] Delete the test tag from remote
git push origin --delete v0.7.15-test-XXXXX

# [ ] Delete the test release from GitHub (if using browser)
#     (Can't delete via CLI, must use GitHub web interface)
```

## Phase 10: Merge to Main

```bash
# [ ] Switch to main branch
git checkout main

# [ ] Ensure main is up to date
git pull origin main

# [ ] Merge the fix branch
git merge fix/homebrew-update-reliability

# [ ] Push to main
git push origin main

# [ ] Verify it's on main
git log -1 --oneline | grep "fix: improve Homebrew"
```

## Phase 11: Real Release Test

Do one final test with a real release:

```bash
# [ ] Create real release tag
git tag v0.7.15

# [ ] Push it
git push origin v0.7.15

# [ ] Monitor Homebrew job in Actions
#     (Should succeed on first try or with 1-2 retries)

# [ ] Verify formula updated in homebrew-howlerops repo
```

## Summary Checklist

- [ ] Read documentation (SUMMARY, VISUAL)
- [ ] Choose implementation method
- [ ] Create backup and branch
- [ ] Apply fixes (Method A/B/C)
- [ ] Verify syntax and changes
- [ ] Stage and commit
- [ ] Test with test tag
- [ ] Monitor workflow
- [ ] Fix any issues or rollback
- [ ] Clean up test artifacts
- [ ] Merge to main
- [ ] Real release test
- [ ] Done!

---

## Key Files for Reference

| File | Purpose | Use When |
|------|---------|----------|
| `HOMEBREW_FIX_SUMMARY.md` | Quick overview | Need executive summary |
| `HOMEBREW_ISSUE_ANALYSIS.md` | Deep technical analysis | Want to understand everything |
| `HOMEBREW_FIXES.md` | Step-by-step implementation | Doing manual edits |
| `APPLY_FIXES.md` | Copy-paste code changes | Want exact code |
| `VISUAL_EXPLANATION.md` | Diagrams and flow charts | Visual learner |
| `update-homebrew-formula-FIXED.sh` | Ready-to-use fixed script | Taking shortcut |
| `IMPLEMENTATION_CHECKLIST.md` | This file | Following along step-by-step |

---

## Time Estimates

- **Understanding phase:** 5-10 minutes
- **Implementation:** 5-15 minutes (depending on method)
- **Verification:** 5 minutes
- **Testing:** 10-15 minutes (wait for workflow to run)
- **Total:** 25-55 minutes

---

## Common Issues & Solutions

### Issue: "bash -n reports syntax error"
**Solution:** Copy-paste error. Check the edits match exactly, or use Method A (fixed script).

### Issue: "Grep doesn't find expected text"
**Solution:** Check file wasn't already modified. Use `git diff` to see what's there.

### Issue: "Test workflow doesn't exist"
**Solution:** Give it a minute, GitHub Actions needs time to pick up the new tag.

### Issue: "Homebrew job still fails after fix"
**Solution:** Check workflow file was edited (sleep 900). May need token investigation.

### Issue: "Can't push tag"
**Solution:** Run `git push origin --tags` (with --tags flag, not just --tag).

---

## Questions?

Before asking for help, check:

1. Did you read `HOMEBREW_ISSUE_ANALYSIS.md`?
2. Did you verify the script syntax with `bash -n`?
3. Did you confirm both files were edited (script + workflow)?
4. Did you test with a test tag first?
5. What does the actual GitHub Actions log show?

