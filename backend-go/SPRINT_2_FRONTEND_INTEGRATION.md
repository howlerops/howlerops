# Sprint 2 Backend Integration Guide for Frontend

## Overview
Sprint 2 added email notifications and rate limiting to organization invitation endpoints. This guide helps frontend integrate these new features.

## API Changes

### CreateInvitation Endpoint

**New Behavior:**
- Now enforces rate limits (20 invitations/hour per user, 5/hour per organization)
- Automatically sends invitation email to the invitee
- May return 429 status code if rate limit exceeded

**Error Responses:**

```typescript
// Rate limit exceeded
{
  "code": "RESOURCE_EXHAUSTED",
  "message": "rate limit exceeded: user rate limit exceeded",
  "details": []
}

// Organization rate limit
{
  "code": "RESOURCE_EXHAUSTED",
  "message": "rate limit exceeded: organization rate limit exceeded",
  "details": []
}
```

**Frontend Changes Needed:**

```typescript
// Handle rate limit errors
async function createInvitation(orgId: string, email: string, role: string) {
  try {
    const response = await api.createInvitation({ orgId, email, role });

    // Show success message
    toast.success(`Invitation sent to ${email}`);

  } catch (error) {
    if (error.code === 'RESOURCE_EXHAUSTED') {
      // Rate limit exceeded
      if (error.message.includes('user rate limit')) {
        toast.error('You\'ve sent too many invitations. Please try again in an hour.');
      } else if (error.message.includes('organization rate limit')) {
        toast.error('Organization invitation limit reached. Please try again in an hour.');
      }
    } else {
      // Other errors
      toast.error('Failed to send invitation');
    }
  }
}
```

### AcceptInvitation Endpoint

**New Behavior:**
- Now sends welcome email automatically after successful acceptance
- No API changes needed

**What Happens:**
1. User clicks "Accept Invitation" in email or UI
2. Backend adds user to organization
3. Backend sends welcome email to user
4. Frontend shows success message

**No frontend changes required** - email is sent automatically.

### RemoveMember Endpoint

**New Behavior:**
- Now sends removal notification email to removed member
- No API changes needed

**What Happens:**
1. Admin/owner removes member
2. Backend removes member from organization
3. Backend sends polite notification email to removed member
4. Frontend shows success message

**No frontend changes required** - email is sent automatically.

## Email Content

### What Users Will Receive

**1. Invitation Email**
- Subject: "You're invited to join [OrgName] on Howlerops"
- Contains:
  - Organization name
  - Inviter name
  - Role being offered
  - "Accept Invitation" button
  - 7-day expiration notice
  - Howlerops branding

**2. Welcome Email (after accepting)**
- Subject: "Welcome to [OrgName]!"
- Contains:
  - Organization name
  - User's role
  - What they can do now
  - "Go to Dashboard" button
  - Getting started tips

**3. Removal Notification**
- Subject: "You've been removed from [OrgName]"
- Contains:
  - Organization name
  - What this means
  - Reassurance about personal data
  - "Create Organization" button

## UI Recommendations

### Invitation Form

Add user feedback about rate limits:

```tsx
function InvitationForm({ orgId }: { orgId: string }) {
  const [email, setEmail] = useState('');
  const [role, setRole] = useState('member');
  const [isLoading, setIsLoading] = useState(false);

  return (
    <form onSubmit={handleSubmit}>
      <input
        type="email"
        value={email}
        onChange={(e) => setEmail(e.target.value)}
        placeholder="colleague@example.com"
      />

      <select value={role} onChange={(e) => setRole(e.target.value)}>
        <option value="member">Member</option>
        <option value="admin">Admin</option>
      </select>

      <button type="submit" disabled={isLoading}>
        {isLoading ? 'Sending...' : 'Send Invitation'}
      </button>

      {/* NEW: Add rate limit info */}
      <p className="text-sm text-gray-500 mt-2">
        You can send up to 20 invitations per hour
      </p>
    </form>
  );
}
```

### Success Messages

Update success messages to mention email:

```typescript
// After creating invitation
toast.success(
  `Invitation sent! ${email} will receive an email with a link to join.`,
  { duration: 5000 }
);

// After accepting invitation
toast.success(
  `Welcome to ${orgName}! Check your email for getting started tips.`
);
```

### Error Handling

Add specific error handling for rate limits:

```tsx
function handleError(error: ApiError) {
  switch (error.code) {
    case 'RESOURCE_EXHAUSTED':
      if (error.message.includes('user rate limit')) {
        return (
          <Alert variant="warning">
            <AlertTitle>Rate Limit Reached</AlertTitle>
            <AlertDescription>
              You've sent 20 invitations in the last hour.
              Please wait before sending more.
            </AlertDescription>
          </Alert>
        );
      } else if (error.message.includes('organization rate limit')) {
        return (
          <Alert variant="warning">
            <AlertTitle>Organization Limit Reached</AlertTitle>
            <AlertDescription>
              This organization has sent 5 invitations in the last hour.
              Please try again later.
            </AlertDescription>
          </Alert>
        );
      }
      break;

    case 'PERMISSION_DENIED':
      return <Alert variant="error">You don't have permission to invite members</Alert>;

    case 'INVALID_ARGUMENT':
      if (error.message.includes('email')) {
        return <Alert variant="error">Please enter a valid email address</Alert>;
      }
      break;

    default:
      return <Alert variant="error">Failed to send invitation. Please try again.</Alert>;
  }
}
```

## Rate Limit Details

### Limits
- **Per User**: 20 invitations per hour
- **Per Organization**: 5 invitations per hour

### How It Works
- Limits are enforced at the API level (backend)
- Token bucket algorithm with 1-hour window
- Both limits must pass for invitation to succeed
- Frontend should handle 429 status codes gracefully

### Recommended Frontend Behavior

```typescript
interface InvitationState {
  sent: number;        // Number sent in last hour
  limit: number;       // Maximum (20)
  remaining: number;   // Remaining quota
  resetAt: Date;       // When quota resets
}

// Optionally track client-side (not enforced, just UX)
function useInvitationQuota(userId: string) {
  const [quota, setQuota] = useState<InvitationState>({
    sent: 0,
    limit: 20,
    remaining: 20,
    resetAt: new Date(Date.now() + 3600000), // 1 hour from now
  });

  const onInvitationSent = () => {
    setQuota(prev => ({
      ...prev,
      sent: prev.sent + 1,
      remaining: Math.max(0, prev.remaining - 1),
    }));
  };

  return { quota, onInvitationSent };
}

// Usage in component
function InvitationList() {
  const { quota, onInvitationSent } = useInvitationQuota(currentUserId);

  return (
    <div>
      <p className="text-sm text-gray-500">
        {quota.remaining} of {quota.limit} invitations remaining this hour
      </p>

      {quota.remaining === 0 && (
        <Alert variant="warning">
          You've reached your hourly limit.
          Resets at {quota.resetAt.toLocaleTimeString()}
        </Alert>
      )}
    </div>
  );
}
```

## Email Template Preview

For better UX, you might want to show users what the invitation email looks like:

```tsx
function InvitationPreview({ orgName, userName }: { orgName: string; userName: string }) {
  return (
    <div className="border rounded-lg p-4 bg-gray-50">
      <h3 className="font-semibold mb-2">Email Preview</h3>
      <div className="bg-white p-4 rounded shadow-sm">
        <div className="text-2xl font-bold text-indigo-600 mb-4">Howlerops</div>
        <h1 className="text-xl font-bold mb-4">You're Invited! 🎉</h1>
        <div className="bg-gradient-to-r from-indigo-500 to-purple-600 text-white p-4 rounded mb-4">
          <div className="font-bold text-lg">{orgName}</div>
        </div>
        <p className="text-gray-600 mb-4">
          <strong>{userName}</strong> has invited you to join their organization
        </p>
        <button className="bg-green-500 text-white px-6 py-3 rounded font-semibold">
          Accept Invitation
        </button>
        <p className="text-sm text-gray-500 mt-4">
          This invitation expires in 7 days
        </p>
      </div>
    </div>
  );
}
```

## Testing

### Manual Testing Checklist

1. **Send Invitation**
   - [ ] Email is sent to invitee
   - [ ] Email contains correct organization name
   - [ ] Email contains correct inviter name
   - [ ] Email contains correct role
   - [ ] "Accept Invitation" button works
   - [ ] Email is mobile-responsive

2. **Rate Limiting**
   - [ ] Can send 20 invitations successfully
   - [ ] 21st invitation fails with clear error
   - [ ] Error message is user-friendly
   - [ ] After 1 hour, can send invitations again

3. **Accept Invitation**
   - [ ] Welcome email is sent
   - [ ] User is added to organization
   - [ ] Welcome email contains correct info
   - [ ] Dashboard link works

4. **Remove Member**
   - [ ] Removal notification is sent
   - [ ] Email is polite and professional
   - [ ] Links to create organization work
   - [ ] Personal data assurance is clear

### Automated Testing

```typescript
// Example test
describe('Invitation Flow', () => {
  it('sends invitation email and shows success message', async () => {
    const { getByRole, getByText } = render(<InvitationForm orgId="org123" />);

    // Fill form
    await userEvent.type(getByRole('textbox', { name: /email/i }), 'test@example.com');
    await userEvent.selectOptions(getByRole('combobox', { name: /role/i }), 'member');

    // Submit
    await userEvent.click(getByRole('button', { name: /send invitation/i }));

    // Check success message mentions email
    await waitFor(() => {
      expect(getByText(/will receive an email/i)).toBeInTheDocument();
    });
  });

  it('shows rate limit error when limit exceeded', async () => {
    // Mock API to return rate limit error
    server.use(
      rest.post('/api/invitations', (req, res, ctx) => {
        return res(
          ctx.status(429),
          ctx.json({
            code: 'RESOURCE_EXHAUSTED',
            message: 'rate limit exceeded: user rate limit exceeded',
          })
        );
      })
    );

    const { getByRole, getByText } = render(<InvitationForm orgId="org123" />);

    await userEvent.type(getByRole('textbox', { name: /email/i }), 'test@example.com');
    await userEvent.click(getByRole('button', { name: /send invitation/i }));

    await waitFor(() => {
      expect(getByText(/too many invitations/i)).toBeInTheDocument();
    });
  });
});
```

## Summary

### What Changed
1. ✅ Invitations now send email automatically
2. ✅ Accepting invitations sends welcome email
3. ✅ Removing members sends notification email
4. ✅ Rate limiting prevents spam (20/hour per user, 5/hour per org)

### What You Need to Do
1. Add rate limit error handling
2. Update success messages to mention email
3. Optionally add rate limit quota display
4. Test email flows manually
5. Update automated tests

### What You Don't Need to Do
1. ❌ No need to manually send emails
2. ❌ No need to track email delivery
3. ❌ No need to implement retry logic
4. ❌ No need to create email templates

All email logic is handled by the backend automatically.
