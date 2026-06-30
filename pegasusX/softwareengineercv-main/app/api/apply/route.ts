import { NextRequest, NextResponse } from 'next/server';
import { Resend } from 'resend';

const resend = process.env.RESEND_API_KEY ? new Resend(process.env.RESEND_API_KEY) : null;

export async function POST(request: NextRequest) {
  try {
    const { name, email, position, portfolio, message } = await request.json();

    // Validate required fields
    if (!name || !email || !position) {
      return NextResponse.json(
        { error: 'Missing required fields' },
        { status: 400 }
      );
    }

    // Create application object
    const application = {
      id: crypto.randomUUID(),
      name,
      email,
      position,
      portfolio,
      message,
      timestamp: new Date().toISOString(),
      read: false
    };

    // Send email notification to you if resend is configured
    let data = null;
    if (resend) {
      data = await resend.emails.send({
      from: 'Team Applications <onboarding@resend.dev>',
      to: [process.env.RECIPIENT_EMAIL || 'your-email@example.com'],
      subject: `New Team Application: ${position} - ${name}`,
      html: `
        <!DOCTYPE html>
        <html>
          <head>
            <style>
              body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
              .container { max-width: 600px; margin: 0 auto; padding: 20px; }
              .header { background: #000; color: #fff; padding: 30px; text-align: center; }
              .content { background: #f4f4f4; padding: 30px; }
              .field { margin-bottom: 20px; }
              .label { font-weight: bold; color: #000; margin-bottom: 5px; }
              .value { background: #fff; padding: 10px; border-left: 4px solid #000; }
              .footer { text-align: center; padding: 20px; color: #666; font-size: 12px; }
              .cta { background: #8DDC96; color: #000; padding: 12px 24px; text-decoration: none; border-radius: 8px; display: inline-block; margin-top: 20px; font-weight: bold; }
            </style>
          </head>
          <body>
            <div class="container">
              <div class="header">
                <h1>🎉 New Team Application</h1>
              </div>
              <div class="content">
                <div class="field">
                  <div class="label">Applicant Name:</div>
                  <div class="value">${name}</div>
                </div>
                <div class="field">
                  <div class="label">Email:</div>
                  <div class="value"><a href="mailto:${email}">${email}</a></div>
                </div>
                <div class="field">
                  <div class="label">Position Applied:</div>
                  <div class="value">${position}</div>
                </div>
                ${portfolio ? `
                <div class="field">
                  <div class="label">Company Website:</div>
                  <div class="value"><a href="${portfolio}" target="_blank">${portfolio}</a></div>
                </div>
                ` : ''}
                ${message ? `
                <div class="field">
                  <div class="label">Message:</div>
                  <div class="value">${message}</div>
                </div>
                ` : ''}
                <div style="text-align: center;">
                  <a href="${process.env.NEXT_PUBLIC_SITE_URL || 'http://localhost:3000'}/admin" class="cta">
                    View in Dashboard →
                  </a>
                </div>
              </div>
              <div class="footer">
                <p>Received on ${new Date().toLocaleString()}</p>
                <p>View all applications in your admin dashboard</p>
              </div>
            </div>
          </body>
        </html>
      `,
      });
    }

    console.log('Application submitted:', { name, email, position, data });

    return NextResponse.json({ 
      success: true, 
      message: 'Application submitted successfully!',
      application 
    });
  } catch (error) {
    console.error('Error sending application:', error);
    return NextResponse.json(
      { error: 'Failed to submit application' },
      { status: 500 }
    );
  }
}
