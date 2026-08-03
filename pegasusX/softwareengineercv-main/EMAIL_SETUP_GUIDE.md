# Email Setup Guide for Team Applications

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



This guide will help you set up email notifications for your team application form so you receive applications directly on your phone.

## 📧 Setup Instructions

### Step 1: Create a Resend Account

1. Go to [https://resend.com](https://resend.com)
2. Sign up for a free account (allows 100 emails/day, 3,000/month)
3. Verify your email address

### Step 2: Get Your API Key

1. Log in to your Resend dashboard
2. Go to **API Keys** section
3. Click **Create API Key**
4. Copy your API key (starts with `re_`)

### Step 3: Configure Environment Variables

1. Create a `.env.local` file in your project root:

```bash
touch .env.local
```

2. Add your credentials to `.env.local`:

```env
RESEND_API_KEY=re_your_actual_api_key_here
RECIPIENT_EMAIL=your-email@example.com
```

**Important:** Replace with your actual values!

### Step 4: Verify Your Domain (Optional but Recommended)

For production, you should verify your own domain:

1. In Resend dashboard, go to **Domains**
2. Click **Add Domain**
3. Enter your domain (e.g., `yourdomain.com`)
4. Add the DNS records provided by Resend to your domain provider
5. Wait for verification (usually 5-10 minutes)
6. Update the `from` field in `/app/api/apply/route.ts`:

```typescript
from: 'Team Applications <applications@yourdomain.com>'
```

### Step 5: Test the Setup

1. Start your development server:

```bash
npm run dev
```

2. Navigate to [http://localhost:3000/join](http://localhost:3000/join)
3. Fill out and submit the application form
4. Check your email inbox for the application notification

## 📱 Receive Emails on Your Phone

### For iOS (iPhone):

1. **Mail App**: Your emails will automatically appear if you use Apple Mail
2. **Gmail App**: Download from App Store if you use Gmail
3. **Enable Push Notifications**:
   - Settings → Mail → Notifications → Allow Notifications
   - Or Settings → Gmail → Notifications → Enable

### For Android:

1. **Gmail App**: Pre-installed or download from Play Store
2. **Enable Notifications**:
   - Settings → Apps → Gmail → Notifications → Enable
   - Ensure "High priority notifications" is enabled

## 🔧 API Endpoint Details

**Endpoint:** `/api/apply`  
**Method:** POST  
**Content-Type:** application/json

**Request Body:**
```json
{
  "name": "John Doe",
  "email": "john@example.com",
  "position": "Frontend Developer",
  "portfolio": "https://portfolio.com",
  "message": "I'd love to join your team!"
}
```

**Success Response (200):**
```json
{
  "success": true,
  "message": "Application submitted successfully!"
}
```

**Error Response (400/500):**
```json
{
  "error": "Error message here"
}
```

## 🎨 Email Template

Applications are sent with a beautiful HTML email template featuring:
- Black and white theme matching your brand
- Applicant's full details
- Clickable email and portfolio links
- Timestamp of submission
- Professional formatting

## 🚀 Deployment Checklist

Before deploying to production:

- [ ] Add `.env.local` to `.gitignore` (already done in Next.js)
- [ ] Set environment variables in your hosting platform:
  - Vercel: Project Settings → Environment Variables
  - Netlify: Site Settings → Build & Deploy → Environment
  - Other platforms: Check their documentation
- [ ] Verify your domain with Resend
- [ ] Update the `from` email address in the API route
- [ ] Test the form on production after deployment

## 🔒 Security Best Practices

1. **Never commit** your `.env.local` file to Git
2. **Use environment variables** for all sensitive data
3. **Add rate limiting** in production to prevent spam:

```typescript
// Consider adding rate limiting middleware
// Example: Use @upstash/ratelimit or similar
```

4. **Add CAPTCHA** for additional protection (optional):
   - Google reCAPTCHA
   - hCaptcha
   - Cloudflare Turnstile

## 📊 Monitoring

Monitor your application submissions:

1. **Resend Dashboard**: View email delivery logs
2. **Server Logs**: Check your hosting platform's logs
3. **Analytics**: Consider adding tracking for form submissions

## 🆘 Troubleshooting

### Emails not arriving?

1. Check spam/junk folder
2. Verify environment variables are set correctly
3. Check Resend dashboard for delivery logs
4. Ensure your API key is valid and not expired
5. Check browser console for errors

### Form not submitting?

1. Open browser developer tools (F12)
2. Check Console for errors
3. Check Network tab for API call status
4. Verify the API route is accessible

### Rate limits exceeded?

- Free plan: 100 emails/day, 3,000/month
- Upgrade to paid plan for higher limits
- Contact Resend support for custom limits

## 💡 Additional Features

Consider adding these enhancements:

1. **Email Confirmation**: Send a confirmation email to applicants
2. **Database Storage**: Store applications in a database (Supabase, MongoDB, etc.)
3. **Admin Dashboard**: Create a page to view all applications
4. **File Uploads**: Allow resume/CV uploads
5. **Status Tracking**: Let applicants track their application status
6. **Auto-responses**: Send automated replies based on position

## 📞 Support

- Resend Documentation: [https://resend.com/docs](https://resend.com/docs)
- Next.js API Routes: [https://nextjs.org/docs/app/building-your-application/routing/route-handlers](https://nextjs.org/docs/app/building-your-application/routing/route-handlers)

---

**Created:** October 12, 2025  
**Last Updated:** October 12, 2025
