# Domain-Based Email Notifications Setup Guide

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



This guide will help you set up email notifications to receive from your own domain instead of the default Resend domain.

## 🌐 Overview

Currently, emails are sent from `onboarding@resend.dev`. To receive emails from your own domain (e.g., `applications@yourdomain.com`), you need to verify your domain with Resend.

## 📋 Prerequisites

1. A domain name you own (e.g., `yourdomain.com`)
2. Access to your domain's DNS settings
3. Resend account (create an API key in the Resend dashboard)

## 🚀 Step-by-Step Setup

### Step 1: Add Your Domain to Resend

1. Go to [Resend Dashboard](https://resend.com/domains)
2. Click **"Add Domain"**
3. Enter your domain (e.g., `yourdomain.com` or subdomain like `mail.yourdomain.com`)
4. Click **"Add"**

### Step 2: Configure DNS Records

Resend will provide you with DNS records to add. You'll need to add these to your domain provider:

**Example DNS Records:**

```
Type: TXT
Name: resend._domainkey
Value: [Resend will provide this]

Type: MX
Name: @
Value: feedback-smtp.us-east-1.amazonses.com
Priority: 10
```

**Where to add DNS records:**

- **Namecheap**: Advanced DNS → Add New Record
- **GoDaddy**: DNS Management → Add Records
- **Cloudflare**: DNS → Add Record
- **Google Domains**: DNS → Custom Records

### Step 3: Verify Domain

1. After adding DNS records, return to Resend dashboard
2. Click **"Verify"** next to your domain
3. Wait 5-10 minutes for DNS propagation
4. Status should change to **"Verified"** ✓

### Step 4: Update Your Code

Once verified, update the email sender addresses in your API routes:

**For Team Applications** (`/app/api/apply/route.ts`):

```typescript
from: 'Team Applications <applications@yourdomain.com>',
```

**For Customer Messages** (`/app/api/contact/route.ts`):

```typescript
from: 'Customer Contact <contact@yourdomain.com>',
```

### Step 5: Update Environment Variables

Add your domain to `.env.local`:

```env
RESEND_API_KEY=re_your_api_key_here
RECIPIENT_EMAIL=you@example.com
NEXT_PUBLIC_SITE_URL=https://yourdomain.com
RESEND_FROM_EMAIL=Pegasus Contact <contact@yourdomain.com>
```

## 📧 Recommended Email Addresses

Set up these email addresses for better organization:

- `applications@yourdomain.com` - For job applications
- `contact@yourdomain.com` - For customer inquiries  
- `hello@yourdomain.com` - General inquiries
- `noreply@yourdomain.com` - Automated emails

## 🔧 Testing Your Setup

After domain verification, test the setup:

1. **Test Application Form**:

   ```
   Visit: http://localhost:3000/join
   Submit a test application
   ```

2. **Test Contact Form**:

   ```
   Visit: http://localhost:3000/contact
   Submit a test message
   ```

3. **Check Email**:
   - Email should arrive at `shsoliyev@aut-edu.uz`
   - Sender should be from your domain
   - Check spam folder if not in inbox

## 📱 Mobile Notifications Setup

### iOS (iPhone)

1. **Settings → Mail → Accounts → Add Account**
2. Select your email provider (Gmail, Outlook, etc.)
3. **Settings → Mail → Notifications → Enable**
4. Enable "Show on Lock Screen" for instant notifications

### Android

1. **Settings → Accounts → Add Account**
2. Select Gmail or your email provider
3. **Settings → Apps → Gmail → Notifications**
4. Enable "Show notifications"
5. Set priority to "High" or "Urgent"

### Gmail Mobile App

1. Open Gmail app
2. **Settings → [Your Account] → Notifications**
3. Enable "All" or "High priority only"
4. Enable vibration and sound

## 🎯 Current Configuration

**Sending Emails:**

- Job Applications: `onboarding@resend.dev` → Your Domain
- Customer Contact: `onboarding@resend.dev` → Your Domain

**Receiving Emails:**

- All notifications go to: `shsoliyev@aut-edu.uz`

**Admin Dashboards:**

- Applications: `http://localhost:3000/admin`
- Messages: `http://localhost:3000/admin/messages`

## 🔒 Security Best Practices

1. **SPF Record**: Resend automatically configures this
2. **DKIM**: Enabled by default with domain verification
3. **DMARC**: Consider adding for enhanced security:

   ```
   Type: TXT
   Name: _dmarc
   Value: v=DMARC1; p=quarantine; rua=mailto:shsoliyev@aut-edu.uz
   ```

## 📊 Email Deliverability Tips

1. **Use a subdomain** (e.g., `mail.yourdomain.com`) to protect main domain reputation
2. **Warm up your domain** by starting with low email volume
3. **Monitor bounce rates** in Resend dashboard
4. **Avoid spam triggers**: Don't use excessive caps, exclamation marks, or spam keywords

## 🆘 Troubleshooting

### Domain Not Verifying?

- Wait 10-15 minutes for DNS propagation
- Use [DNS Checker](https://dnschecker.org) to verify records are live
- Ensure exact values from Resend (no extra spaces)

### Emails Going to Spam?

- Domain must be verified
- Add DMARC record
- Ask recipients to mark as "Not Spam"
- Ensure proper HTML email formatting

### Not Receiving Notifications on Phone?

- Check email app notification settings
- Verify email is syncing (refresh inbox)
- Check phone's "Do Not Disturb" mode
- Ensure background data is enabled for email app

## 💡 Pro Tips

1. **Set up email forwarding** from your custom domain to your personal email
2. **Create email aliases** for different purposes
3. **Use email filters** to organize applications and messages
4. **Enable two-factor authentication** on your email account
5. **Backup important applications** regularly from the admin dashboard

## 📞 Support Resources

- **Resend Documentation**: [resend.com/docs/send-with-nextjs](https://resend.com/docs/send-with-nextjs)
- **DNS Help**: Contact your domain provider's support
- **Email Issues**: Check Resend dashboard logs

## 🎨 Custom Email Templates

Your emails use beautiful HTML templates with your brand colors:

- **Applications**: Green theme (#8DDC96)
- **Messages**: Blue theme (#A9EBF9)
- Both match your website's black & white design system

---

**Current Status:**

- ✅ Logo set to atom.jpeg
- ✅ Email API configured
- ✅ Admin dashboards created
- ⏳ Domain verification (follow steps above)

**Last Updated:** October 12, 2025
