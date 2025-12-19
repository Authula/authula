package util

import (
	"fmt"

	"github.com/GoBetterAuth/go-better-auth/models"
)

// CreateVerificationEmailBody creates the HTML body for an email verification email
func CreateVerificationEmailBody(user models.User, verificationURL string) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; background: #f9f9f9; border-radius: 5px; }
        .button { display: inline-block; padding: 12px 24px; background: #28a745; color: white; text-decoration: none; border-radius: 5px; margin-top: 15px; }
        .footer { margin-top: 20px; padding-top: 20px; border-top: 1px solid #ddd; font-size: 0.9em; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <h2>Verify Your Email Address</h2>
        <p>Hello %s,</p>
        <p>Thank you for signing up! Please verify your email address by clicking the button below:</p>
        <a href="%s" class="button">Verify Email</a>
        <p>This link will expire in 24 hours.</p>
        <p>If you didn't sign up for this account, you can safely ignore this email.</p>
        <div class="footer">
            <p>Best regards,<br>The GoBetterAuth Team</p>
        </div>
    </div>
</body>
</html>
`, user.Name, verificationURL)
}

// CreateResetPasswordEmailBody creates the HTML body for a password reset email
func CreateResetPasswordEmailBody(user models.User, resetURL string) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; background: #f9f9f9; border-radius: 5px; }
        .button { display: inline-block; padding: 12px 24px; background: #007bff; color: white; text-decoration: none; border-radius: 5px; margin-top: 15px; }
        .footer { margin-top: 20px; padding-top: 20px; border-top: 1px solid #ddd; font-size: 0.9em; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <h2>Password Reset Request</h2>
        <p>Hello %s,</p>
        <p>We received a request to reset your password. Click the button below to create a new password:</p>
        <a href="%s" class="button">Reset Password</a>
        <p>This link will expire in 24 hours.</p>
        <p>If you didn't request this reset, you can safely ignore this email.</p>
        <div class="footer">
            <p>Best regards,<br>The GoBetterAuth Team</p>
        </div>
    </div>
</body>
</html>
`, user.Name, resetURL)
}

// CreateEmailChangeVerificationBody creates the HTML body for an email change verification email
func CreateEmailChangeVerificationBody(user models.User, newEmail string, verificationURL string) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; background: #f9f9f9; border-radius: 5px; }
        .button { display: inline-block; padding: 12px 24px; background: #17a2b8; color: white; text-decoration: none; border-radius: 5px; margin-top: 15px; }
        .footer { margin-top: 20px; padding-top: 20px; border-top: 1px solid #ddd; font-size: 0.9em; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <h2>Confirm Your New Email Address</h2>
        <p>Hello %s,</p>
        <p>A request has been made to change your email address to <strong>%s</strong>. Please confirm this change by clicking the button below:</p>
        <a href="%s" class="button">Confirm Email Change</a>
        <p>This link will expire in 24 hours.</p>
        <p>If you didn't request this change, please ignore this email and your current email address will remain unchanged.</p>
        <div class="footer">
            <p>Best regards,<br>The GoBetterAuth Team</p>
        </div>
    </div>
</body>
</html>
`, user.Name, newEmail, verificationURL)
}
