import { NextRequest, NextResponse } from 'next/server';
import { Resend } from 'resend';

const resend = process.env.RESEND_API_KEY ? new Resend(process.env.RESEND_API_KEY) : null;

// Basic email format validator regex
const EMAIL_REGEX = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export async function POST(request: NextRequest) {
  try {
    const body = await request.json();
    const { email } = body || {};

    if (!email || typeof email !== 'string' || !EMAIL_REGEX.test(email.trim())) {
      return NextResponse.json(
        { error: 'Please enter a valid email address.' },
        { status: 400 }
      );
    }

    const cleanEmail = email.trim().toLowerCase();

    // Log subscription event internally
    console.log(`[SUBSCRIBE] New newsletter subscriber: ${cleanEmail} at ${new Date().toISOString()}`);

    // If Resend API key is configured, send confirmation/notification email
    let emailSent = false;
    if (resend) {
      try {
        await resend.emails.send({
          from: 'Pegasus Logistics <onboarding@resend.dev>',
          to: [process.env.RECIPIENT_EMAIL || 'shsoliyev@aut-edu.uz'],
          replyTo: cleanEmail,
          subject: '⚡ New Pegasus Newsletter Subscription',
          html: `
            <div style="font-family: sans-serif; background: #0a0a0a; color: #fff; padding: 30px; border-radius: 12px; border: 1px solid #222;">
              <h2 style="color: #10b981; margin-top: 0;">⚡ New Newsletter Subscriber</h2>
              <p>A new user has subscribed to Pegasus logistics updates:</p>
              <div style="background: #161616; padding: 16px; border-left: 4px solid #10b981; border-radius: 6px; font-family: monospace; font-size: 15px;">
                ${cleanEmail}
              </div>
              <p style="color: #888; font-size: 12px; margin-top: 24px;">
                Timestamp: ${new Date().toLocaleString()}
              </p>
            </div>
          `,
        });
        emailSent = true;
      } catch (emailErr) {
        console.error('[SUBSCRIBE] Failed to send email via Resend:', emailErr);
      }
    }

    return NextResponse.json({
      success: true,
      message: 'Subscribed successfully',
      email: cleanEmail,
      emailSent,
      timestamp: new Date().toISOString(),
    });
  } catch (err) {
    console.error('[SUBSCRIBE] Error handling subscription:', err);
    return NextResponse.json(
      { error: 'Server error processing subscription. Please try again.' },
      { status: 500 }
    );
  }
}
