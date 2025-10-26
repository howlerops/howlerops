package email

// verificationTemplate is the HTML template for email verification
const verificationTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Verify Your Email</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 600px;
            margin: 0 auto;
            padding: 20px;
            background-color: #f4f4f4;
        }
        .container {
            background-color: #ffffff;
            border-radius: 8px;
            padding: 40px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .header {
            text-align: center;
            margin-bottom: 30px;
        }
        .logo {
            font-size: 32px;
            font-weight: bold;
            color: #4F46E5;
            margin-bottom: 10px;
        }
        h1 {
            color: #1F2937;
            font-size: 24px;
            margin-bottom: 20px;
        }
        p {
            color: #6B7280;
            margin-bottom: 20px;
        }
        .button {
            display: inline-block;
            padding: 14px 32px;
            background-color: #4F46E5;
            color: #ffffff;
            text-decoration: none;
            border-radius: 6px;
            font-weight: 600;
            margin: 20px 0;
        }
        .button:hover {
            background-color: #4338CA;
        }
        .token-box {
            background-color: #F3F4F6;
            border: 1px solid #E5E7EB;
            border-radius: 6px;
            padding: 16px;
            margin: 20px 0;
            text-align: center;
            font-family: 'Courier New', monospace;
            font-size: 18px;
            letter-spacing: 2px;
            color: #1F2937;
        }
        .footer {
            margin-top: 40px;
            text-align: center;
            font-size: 14px;
            color: #9CA3AF;
            border-top: 1px solid #E5E7EB;
            padding-top: 20px;
        }
        .warning {
            background-color: #FEF3C7;
            border-left: 4px solid #F59E0B;
            padding: 12px 16px;
            margin: 20px 0;
            border-radius: 4px;
            font-size: 14px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <div class="logo">SQL Studio</div>
        </div>

        <h1>Verify Your Email Address</h1>

        <p>Hello,</p>

        <p>Thank you for signing up for SQL Studio! To complete your registration, please verify your email address by clicking the button below:</p>

        <div style="text-align: center;">
            <a href="{{.VerificationURL}}" class="button">Verify Email Address</a>
        </div>

        <p>Or copy and paste this link into your browser:</p>

        <div class="token-box">
            {{.VerificationURL}}
        </div>

        <div class="warning">
            <strong>Important:</strong> This verification link will expire in 24 hours. If you didn't create an account with SQL Studio, you can safely ignore this email.
        </div>

        <p>If you have any questions or need assistance, please don't hesitate to reach out to our support team.</p>

        <p>Best regards,<br>The SQL Studio Team</p>

        <div class="footer">
            <p>&copy; {{.Year}} SQL Studio. All rights reserved.</p>
            <p>This is an automated message, please do not reply to this email.</p>
        </div>
    </div>
</body>
</html>
`

// passwordResetTemplate is the HTML template for password reset
const passwordResetTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Reset Your Password</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 600px;
            margin: 0 auto;
            padding: 20px;
            background-color: #f4f4f4;
        }
        .container {
            background-color: #ffffff;
            border-radius: 8px;
            padding: 40px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .header {
            text-align: center;
            margin-bottom: 30px;
        }
        .logo {
            font-size: 32px;
            font-weight: bold;
            color: #4F46E5;
            margin-bottom: 10px;
        }
        h1 {
            color: #1F2937;
            font-size: 24px;
            margin-bottom: 20px;
        }
        p {
            color: #6B7280;
            margin-bottom: 20px;
        }
        .button {
            display: inline-block;
            padding: 14px 32px;
            background-color: #DC2626;
            color: #ffffff;
            text-decoration: none;
            border-radius: 6px;
            font-weight: 600;
            margin: 20px 0;
        }
        .button:hover {
            background-color: #B91C1C;
        }
        .token-box {
            background-color: #F3F4F6;
            border: 1px solid #E5E7EB;
            border-radius: 6px;
            padding: 16px;
            margin: 20px 0;
            text-align: center;
            font-family: 'Courier New', monospace;
            font-size: 18px;
            letter-spacing: 2px;
            color: #1F2937;
        }
        .footer {
            margin-top: 40px;
            text-align: center;
            font-size: 14px;
            color: #9CA3AF;
            border-top: 1px solid #E5E7EB;
            padding-top: 20px;
        }
        .warning {
            background-color: #FEE2E2;
            border-left: 4px solid #DC2626;
            padding: 12px 16px;
            margin: 20px 0;
            border-radius: 4px;
            font-size: 14px;
        }
        .info {
            background-color: #DBEAFE;
            border-left: 4px solid #3B82F6;
            padding: 12px 16px;
            margin: 20px 0;
            border-radius: 4px;
            font-size: 14px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <div class="logo">SQL Studio</div>
        </div>

        <h1>Reset Your Password</h1>

        <p>Hello,</p>

        <p>We received a request to reset the password for your SQL Studio account. Click the button below to create a new password:</p>

        <div style="text-align: center;">
            <a href="{{.ResetURL}}" class="button">Reset Password</a>
        </div>

        <p>Or copy and paste this link into your browser:</p>

        <div class="token-box">
            {{.ResetURL}}
        </div>

        <div class="warning">
            <strong>Security Notice:</strong> This password reset link will expire in 1 hour. If you didn't request a password reset, please ignore this email and your password will remain unchanged.
        </div>

        <div class="info">
            <strong>Tip:</strong> To keep your account secure, choose a strong password that:
            <ul style="margin: 10px 0 0 20px;">
                <li>Is at least 8 characters long</li>
                <li>Contains uppercase and lowercase letters</li>
                <li>Includes numbers and special characters</li>
                <li>Is unique to SQL Studio</li>
            </ul>
        </div>

        <p>If you're having trouble or didn't request this reset, please contact our support team immediately.</p>

        <p>Best regards,<br>The SQL Studio Team</p>

        <div class="footer">
            <p>&copy; {{.Year}} SQL Studio. All rights reserved.</p>
            <p>This is an automated message, please do not reply to this email.</p>
        </div>
    </div>
</body>
</html>
`

// welcomeTemplate is the HTML template for welcome emails
const welcomeTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Welcome to SQL Studio</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 600px;
            margin: 0 auto;
            padding: 20px;
            background-color: #f4f4f4;
        }
        .container {
            background-color: #ffffff;
            border-radius: 8px;
            padding: 40px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .header {
            text-align: center;
            margin-bottom: 30px;
        }
        .logo {
            font-size: 32px;
            font-weight: bold;
            color: #4F46E5;
            margin-bottom: 10px;
        }
        h1 {
            color: #1F2937;
            font-size: 28px;
            margin-bottom: 20px;
            text-align: center;
        }
        h2 {
            color: #1F2937;
            font-size: 20px;
            margin-top: 30px;
            margin-bottom: 15px;
        }
        p {
            color: #6B7280;
            margin-bottom: 20px;
        }
        .button {
            display: inline-block;
            padding: 14px 32px;
            background-color: #4F46E5;
            color: #ffffff;
            text-decoration: none;
            border-radius: 6px;
            font-weight: 600;
            margin: 20px 0;
        }
        .button:hover {
            background-color: #4338CA;
        }
        .features {
            background-color: #F9FAFB;
            border-radius: 6px;
            padding: 24px;
            margin: 30px 0;
        }
        .feature-item {
            margin-bottom: 16px;
            padding-left: 24px;
            position: relative;
        }
        .feature-item:before {
            content: "✓";
            position: absolute;
            left: 0;
            color: #10B981;
            font-weight: bold;
            font-size: 18px;
        }
        .footer {
            margin-top: 40px;
            text-align: center;
            font-size: 14px;
            color: #9CA3AF;
            border-top: 1px solid #E5E7EB;
            padding-top: 20px;
        }
        .cta-box {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            border-radius: 8px;
            padding: 24px;
            text-align: center;
            margin: 30px 0;
        }
        .cta-box h3 {
            color: #ffffff;
            margin-bottom: 10px;
        }
        .cta-box p {
            color: #E0E7FF;
            margin-bottom: 20px;
        }
        .button-white {
            display: inline-block;
            padding: 14px 32px;
            background-color: #ffffff;
            color: #4F46E5;
            text-decoration: none;
            border-radius: 6px;
            font-weight: 600;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <div class="logo">SQL Studio</div>
        </div>

        <h1>Welcome to SQL Studio! 🎉</h1>

        <p>Hi {{.Name}},</p>

        <p>We're thrilled to have you on board! SQL Studio is your powerful companion for working with databases, and we can't wait for you to explore all its features.</p>

        <div class="features">
            <h2>What you can do with SQL Studio:</h2>

            <div class="feature-item">
                <strong>Connect to Multiple Databases</strong> - MySQL, PostgreSQL, SQLite, MongoDB, and more
            </div>

            <div class="feature-item">
                <strong>Smart Query Editor</strong> - Syntax highlighting, auto-completion, and multi-query execution
            </div>

            <div class="feature-item">
                <strong>AI-Powered Assistance</strong> - Get help writing queries and understanding your data
            </div>

            <div class="feature-item">
                <strong>Cloud Sync</strong> - Access your connections and saved queries from anywhere
            </div>

            <div class="feature-item">
                <strong>Secure Storage</strong> - Encrypted connection credentials and local-first architecture
            </div>

            <div class="feature-item">
                <strong>Beautiful Visualizations</strong> - Turn your data into insights with charts and graphs
            </div>
        </div>

        <div class="cta-box">
            <h3>Ready to Get Started?</h3>
            <p>Launch SQL Studio and create your first database connection</p>
            <a href="https://sqlstudio.io/docs/getting-started" class="button-white">View Getting Started Guide</a>
        </div>

        <h2>Need Help?</h2>

        <p>We're here to help you succeed! Here are some resources to get you started:</p>

        <ul style="color: #6B7280;">
            <li><strong>Documentation:</strong> Comprehensive guides and tutorials</li>
            <li><strong>Community:</strong> Join our Discord community for support</li>
            <li><strong>Support:</strong> Email us at support@sqlstudio.io</li>
        </ul>

        <p>We're constantly improving SQL Studio based on user feedback. If you have any suggestions or run into any issues, please don't hesitate to reach out!</p>

        <p>Happy querying!</p>

        <p>Best regards,<br>The SQL Studio Team</p>

        <div class="footer">
            <p>&copy; {{.Year}} SQL Studio. All rights reserved.</p>
            <p>You're receiving this email because you created an account at SQL Studio.</p>
        </div>
    </div>
</body>
</html>
`

// organizationInvitationTemplate is the HTML template for organization invitations
const organizationInvitationTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>You're Invited to Join an Organization</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 600px;
            margin: 0 auto;
            padding: 20px;
            background-color: #f4f4f4;
        }
        .container {
            background-color: #ffffff;
            border-radius: 8px;
            padding: 40px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .header {
            text-align: center;
            margin-bottom: 30px;
        }
        .logo {
            font-size: 32px;
            font-weight: bold;
            color: #4F46E5;
            margin-bottom: 10px;
        }
        h1 {
            color: #1F2937;
            font-size: 26px;
            margin-bottom: 20px;
            text-align: center;
        }
        .org-badge {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            font-size: 20px;
            font-weight: bold;
            padding: 16px 24px;
            border-radius: 8px;
            text-align: center;
            margin: 24px 0;
        }
        .role-badge {
            display: inline-block;
            background-color: #DBEAFE;
            color: #1E40AF;
            padding: 6px 12px;
            border-radius: 4px;
            font-size: 14px;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }
        p {
            color: #6B7280;
            margin-bottom: 20px;
        }
        .button {
            display: inline-block;
            padding: 16px 40px;
            background-color: #10B981;
            color: #ffffff;
            text-decoration: none;
            border-radius: 6px;
            font-weight: 600;
            font-size: 16px;
            margin: 20px 0;
        }
        .button:hover {
            background-color: #059669;
        }
        .info-box {
            background-color: #F0FDF4;
            border-left: 4px solid #10B981;
            padding: 16px;
            margin: 24px 0;
            border-radius: 4px;
        }
        .info-box h3 {
            color: #065F46;
            margin: 0 0 8px 0;
            font-size: 16px;
        }
        .info-box p {
            margin: 4px 0;
            font-size: 14px;
            color: #047857;
        }
        .warning {
            background-color: #FEF3C7;
            border-left: 4px solid #F59E0B;
            padding: 12px 16px;
            margin: 20px 0;
            border-radius: 4px;
            font-size: 14px;
        }
        .footer {
            margin-top: 40px;
            text-align: center;
            font-size: 14px;
            color: #9CA3AF;
            border-top: 1px solid #E5E7EB;
            padding-top: 20px;
        }
        .inviter-info {
            background-color: #F9FAFB;
            padding: 16px;
            border-radius: 6px;
            margin: 20px 0;
            text-align: center;
        }
        .inviter-info strong {
            color: #1F2937;
            font-size: 16px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <div class="logo">SQL Studio</div>
        </div>

        <h1>You're Invited! 🎉</h1>

        <div class="org-badge">
            {{.OrgName}}
        </div>

        <div class="inviter-info">
            <strong>{{.InviterName}}</strong> has invited you to join their organization
        </div>

        <p>You've been invited to join <strong>{{.OrgName}}</strong> on SQL Studio as a <span class="role-badge">{{.Role}}</span>.</p>

        <div class="info-box">
            <h3>What you'll get:</h3>
            <p>✓ Shared database connections with your team</p>
            <p>✓ Collaborate on SQL queries and schemas</p>
            <p>✓ Access to organization resources and saved queries</p>
            <p>✓ Real-time collaboration features</p>
        </div>

        <div style="text-align: center;">
            <a href="{{.InvitationURL}}" class="button">Accept Invitation</a>
        </div>

        <p style="text-align: center; margin-top: 16px; font-size: 14px;">
            Or copy and paste this link into your browser:<br>
            <span style="word-break: break-all; color: #6B7280; font-family: monospace; font-size: 12px;">{{.InvitationURL}}</span>
        </p>

        <div class="warning">
            <strong>Important:</strong> This invitation link will expire in 7 days. If you don't want to join this organization, you can safely ignore this email.
        </div>

        <p>If you have any questions about this invitation, please contact <strong>{{.InviterName}}</strong> or reach out to our support team.</p>

        <p>Best regards,<br>The SQL Studio Team</p>

        <div class="footer">
            <p>&copy; {{.Year}} SQL Studio. All rights reserved.</p>
            <p>This is an automated message, please do not reply to this email.</p>
        </div>
    </div>
</body>
</html>
`

// organizationWelcomeTemplate is the HTML template for organization welcome emails
const organizationWelcomeTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Welcome to the Organization</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 600px;
            margin: 0 auto;
            padding: 20px;
            background-color: #f4f4f4;
        }
        .container {
            background-color: #ffffff;
            border-radius: 8px;
            padding: 40px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .header {
            text-align: center;
            margin-bottom: 30px;
        }
        .logo {
            font-size: 32px;
            font-weight: bold;
            color: #4F46E5;
            margin-bottom: 10px;
        }
        h1 {
            color: #1F2937;
            font-size: 28px;
            margin-bottom: 20px;
            text-align: center;
        }
        h2 {
            color: #1F2937;
            font-size: 20px;
            margin-top: 30px;
            margin-bottom: 15px;
        }
        .org-badge {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            font-size: 20px;
            font-weight: bold;
            padding: 16px 24px;
            border-radius: 8px;
            text-align: center;
            margin: 24px 0;
        }
        .role-badge {
            display: inline-block;
            background-color: #DBEAFE;
            color: #1E40AF;
            padding: 8px 16px;
            border-radius: 4px;
            font-size: 14px;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 0.5px;
            margin: 8px 0;
        }
        p {
            color: #6B7280;
            margin-bottom: 20px;
        }
        .button {
            display: inline-block;
            padding: 14px 32px;
            background-color: #4F46E5;
            color: #ffffff;
            text-decoration: none;
            border-radius: 6px;
            font-weight: 600;
            margin: 20px 0;
        }
        .button:hover {
            background-color: #4338CA;
        }
        .features {
            background-color: #F9FAFB;
            border-radius: 6px;
            padding: 24px;
            margin: 30px 0;
        }
        .feature-item {
            margin-bottom: 16px;
            padding-left: 24px;
            position: relative;
        }
        .feature-item:before {
            content: "✓";
            position: absolute;
            left: 0;
            color: #10B981;
            font-weight: bold;
            font-size: 18px;
        }
        .footer {
            margin-top: 40px;
            text-align: center;
            font-size: 14px;
            color: #9CA3AF;
            border-top: 1px solid #E5E7EB;
            padding-top: 20px;
        }
        .cta-box {
            background: linear-gradient(135deg, #10B981 0%, #059669 100%);
            border-radius: 8px;
            padding: 24px;
            text-align: center;
            margin: 30px 0;
        }
        .cta-box h3 {
            color: #ffffff;
            margin-bottom: 10px;
        }
        .cta-box p {
            color: #D1FAE5;
            margin-bottom: 20px;
        }
        .button-white {
            display: inline-block;
            padding: 14px 32px;
            background-color: #ffffff;
            color: #10B981;
            text-decoration: none;
            border-radius: 6px;
            font-weight: 600;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <div class="logo">SQL Studio</div>
        </div>

        <h1>Welcome! 🎉</h1>

        <div class="org-badge">
            {{.OrgName}}
        </div>

        <p style="text-align: center;">
            Your role: <span class="role-badge">{{.Role}}</span>
        </p>

        <p>Hi {{.Name}},</p>

        <p>Welcome to <strong>{{.OrgName}}</strong>! You're now part of the team and can start collaborating on database projects right away.</p>

        <div class="features">
            <h2>What you can do now:</h2>

            <div class="feature-item">
                <strong>Access Shared Connections</strong> - Connect to your team's databases
            </div>

            <div class="feature-item">
                <strong>Collaborate on Queries</strong> - Share and reuse SQL queries with your team
            </div>

            <div class="feature-item">
                <strong>View Team Activity</strong> - See what your teammates are working on
            </div>

            <div class="feature-item">
                <strong>Real-time Sync</strong> - Your work syncs automatically across devices
            </div>

            <div class="feature-item">
                <strong>Secure by Default</strong> - All data is encrypted and access-controlled
            </div>
        </div>

        <div class="cta-box">
            <h3>Ready to Get Started?</h3>
            <p>Head over to your organization dashboard</p>
            <a href="https://sqlstudio.io/dashboard" class="button-white">Go to Dashboard</a>
        </div>

        <h2>Getting Started Tips:</h2>

        <ul style="color: #6B7280;">
            <li>Explore your organization's shared database connections</li>
            <li>Check out the saved queries library</li>
            <li>Set up your profile and preferences</li>
            <li>Join your team's collaboration channels</li>
        </ul>

        <p>If you have any questions or need help getting started, don't hesitate to reach out to your team or contact our support team at support@sqlstudio.io.</p>

        <p>Happy collaborating!</p>

        <p>Best regards,<br>The SQL Studio Team</p>

        <div class="footer">
            <p>&copy; {{.Year}} SQL Studio. All rights reserved.</p>
            <p>You're receiving this email because you joined {{.OrgName}} on SQL Studio.</p>
        </div>
    </div>
</body>
</html>
`

// memberRemovedTemplate is the HTML template for member removal notifications
const memberRemovedTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Organization Membership Update</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 600px;
            margin: 0 auto;
            padding: 20px;
            background-color: #f4f4f4;
        }
        .container {
            background-color: #ffffff;
            border-radius: 8px;
            padding: 40px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .header {
            text-align: center;
            margin-bottom: 30px;
        }
        .logo {
            font-size: 32px;
            font-weight: bold;
            color: #4F46E5;
            margin-bottom: 10px;
        }
        h1 {
            color: #1F2937;
            font-size: 24px;
            margin-bottom: 20px;
            text-align: center;
        }
        p {
            color: #6B7280;
            margin-bottom: 20px;
        }
        .org-badge {
            background-color: #F3F4F6;
            border: 2px solid #E5E7EB;
            color: #1F2937;
            font-size: 18px;
            font-weight: bold;
            padding: 16px 24px;
            border-radius: 8px;
            text-align: center;
            margin: 24px 0;
        }
        .button {
            display: inline-block;
            padding: 14px 32px;
            background-color: #4F46E5;
            color: #ffffff;
            text-decoration: none;
            border-radius: 6px;
            font-weight: 600;
            margin: 20px 0;
        }
        .button:hover {
            background-color: #4338CA;
        }
        .info-box {
            background-color: #EFF6FF;
            border-left: 4px solid #3B82F6;
            padding: 16px;
            margin: 24px 0;
            border-radius: 4px;
        }
        .info-box h3 {
            color: #1E40AF;
            margin: 0 0 8px 0;
            font-size: 16px;
        }
        .info-box p {
            margin: 4px 0;
            font-size: 14px;
            color: #1E40AF;
        }
        .footer {
            margin-top: 40px;
            text-align: center;
            font-size: 14px;
            color: #9CA3AF;
            border-top: 1px solid #E5E7EB;
            padding-top: 20px;
        }
        .cta-box {
            background-color: #F9FAFB;
            border-radius: 8px;
            padding: 24px;
            text-align: center;
            margin: 30px 0;
        }
        .cta-box h3 {
            color: #1F2937;
            margin-bottom: 10px;
        }
        .cta-box p {
            color: #6B7280;
            margin-bottom: 20px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <div class="logo">SQL Studio</div>
        </div>

        <h1>Organization Membership Update</h1>

        <div class="org-badge">
            {{.OrgName}}
        </div>

        <p>Hello,</p>

        <p>This is to inform you that you have been removed from the organization <strong>{{.OrgName}}</strong> on SQL Studio.</p>

        <div class="info-box">
            <h3>What this means:</h3>
            <p>• You no longer have access to this organization's resources</p>
            <p>• Shared database connections are no longer available to you</p>
            <p>• Organization queries and data are no longer accessible</p>
            <p>• Your personal data and connections remain intact</p>
        </div>

        <p>Your personal SQL Studio account remains active, and all your individual connections, queries, and data are safe and accessible.</p>

        <div class="cta-box">
            <h3>Want to create your own organization?</h3>
            <p>You can create and manage your own organizations on SQL Studio</p>
            <a href="https://sqlstudio.io/organizations/new" class="button">Create Organization</a>
        </div>

        <p>If you believe this was done in error or have questions about this change, please contact the organization administrator or our support team at support@sqlstudio.io.</p>

        <p>Thank you for using SQL Studio.</p>

        <p>Best regards,<br>The SQL Studio Team</p>

        <div class="footer">
            <p>&copy; {{.Year}} SQL Studio. All rights reserved.</p>
            <p>This is an automated notification. Please do not reply to this email.</p>
        </div>
    </div>
</body>
</html>
`
