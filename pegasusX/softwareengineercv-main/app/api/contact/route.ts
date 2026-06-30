import { NextRequest, NextResponse } from 'next/server';
import { Resend } from 'resend';

const resend = process.env.RESEND_API_KEY ? new Resend(process.env.RESEND_API_KEY) : null;

export async function POST(request: NextRequest) {
  try {
    const { name, email, company, budget, timeline, category, message, inquiryType } = await request.json();

    // Validate required fields
    if (!name || !email || !message) {
      return NextResponse.json(
        { error: 'Missing required fields' },
        { status: 400 }
      );
    }

    // Create inquiry object
    const inquiry = {
      id: crypto.randomUUID(),
      name,
      email,
      company: company || '',
      budget: budget || '',
      timeline: timeline || '',
      category: category || '',
      message,
      inquiryType: inquiryType || 'general',
      timestamp: new Date().toISOString(),
      read: false
    };

    // Determine email subject based on inquiry type
    const subjectMap = {
      general: 'New General Inquiry',
      client: 'New Client Project Inquiry',
      sponsor: 'New Sponsorship Inquiry'
    };
    const emailSubject = subjectMap[inquiryType as keyof typeof subjectMap] || 'New Contact Message';

    // Build email content based on inquiry type
    let additionalFields = '';
    
    if (inquiryType === 'client') {
      additionalFields = `
        ${company ? `<div class="field"><div class="label">Company:</div><div class="value">${company}</div></div>` : ''}
        ${budget ? `<div class="field"><div class="label">Project Budget:</div><div class="value">${budget}</div></div>` : ''}
        ${timeline ? `<div class="field"><div class="label">Timeline:</div><div class="value">${timeline}</div></div>` : ''}
      `;
    } else if (inquiryType === 'sponsor') {
      additionalFields = `
        ${company ? `<div class="field"><div class="label">Company:</div><div class="value">${company}</div></div>` : ''}
        ${category ? `<div class="field"><div class="label">Sponsorship Type:</div><div class="value">${category}</div></div>` : ''}
      `;
    }

    // Send email notification if resend is configured
    let data = null;
    if (resend) {
      data = await resend.emails.send({
      from: 'Pegasus Contact <onboarding@resend.dev>',
      to: [process.env.RECIPIENT_EMAIL || 'shsoliyev@aut-edu.uz'],
      replyTo: email,
      subject: emailSubject,
      html: `
        <!DOCTYPE html>
        <html>
          <head>
            <style>
              body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Arial, sans-serif; line-height: 1.6; color: #000; margin: 0; padding: 0; }
              .container { max-width: 600px; margin: 0 auto; background: #fff; }
              .header { background: #000; color: #fff; padding: 40px 30px; text-align: center; border-bottom: 2px solid #fff; }
              .header h1 { margin: 0; font-size: 28px; font-weight: bold; }
              .badge { display: inline-block; background: #A9EBF9; color: #000; padding: 8px 16px; border-radius: 20px; font-size: 12px; font-weight: bold; margin-top: 10px; }
              .content { padding: 40px 30px; background: #f8f8f8; }
              .field { margin-bottom: 24px; }
              .label { font-weight: bold; color: #000; margin-bottom: 8px; font-size: 12px; text-transform: uppercase; letter-spacing: 0.5px; }
              .value { background: #fff; padding: 16px; border-left: 4px solid #000; border-radius: 8px; font-size: 14px; }
              .message-box { background: #fff; padding: 20px; border: 2px solid #000; border-radius: 12px; margin-top: 10px; white-space: pre-wrap; }
              .cta { background: #000; color: #fff; padding: 16px 32px; text-decoration: none; border-radius: 12px; display: inline-block; margin-top: 30px; font-weight: bold; border: 2px solid #000; transition: all 0.3s; }
              .cta:hover { background: #FBFF63; color: #000; }
              .footer { text-align: center; padding: 30px; background: #000; color: #fff; font-size: 13px; }
              .footer a { color: #A9EBF9; text-decoration: none; }
              .divider { height: 2px; background: #000; margin: 30px 0; }
            </style>
          </head>
          <body>
            <div class="container">
              <div class="header">
                <h1>📩 ${emailSubject}</h1>
                <span class="badge">${inquiryType.toUpperCase()}</span>
              </div>
              
              <div class="content">
                <div class="field">
                  <div class="label">Name</div>
                  <div class="value">${name}</div>
                </div>
                
                <div class="field">
                  <div class="label">Email</div>
                  <div class="value"><a href="mailto:${email}" style="color: #000; text-decoration: underline;">${email}</a></div>
                </div>
                
                ${additionalFields}
                
                <div class="field">
                  <div class="label">Message</div>
                  <div class="message-box">${message}</div>
                </div>
                
                <div class="divider"></div>
                
                <div style="text-align: center;">
                  <a href="${process.env.NEXT_PUBLIC_SITE_URL || 'http://localhost:3000'}/admin/messages" class="cta">
                    View in Dashboard →
                  </a>
                </div>
              </div>
              
              <div class="footer">
                <p style="margin: 0 0 10px 0;"><strong>Received:</strong> ${new Date().toLocaleString('en-US', { dateStyle: 'full', timeStyle: 'short' })}</p>
                <p style="margin: 0;">Reply directly to: <a href="mailto:${email}">${email}</a></p>
              </div>
            </div>
          </body>
        </html>
      `,
      });
    }

    console.log('Contact inquiry submitted:', { name, email, inquiryType, data });

    return NextResponse.json({ 
      success: true,
      inquiry,
      emailSent: true
    });
  } catch (error) {
    console.error('Error sending contact inquiry:', error);
    return NextResponse.json(
      { error: 'Failed to send inquiry. Please try again or email directly.' },
      { status: 500 }
    );
  }
}
