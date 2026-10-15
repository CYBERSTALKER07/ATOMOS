import { NextRequest, NextResponse } from 'next/server';
import {
  fetchEnterpriseRequests,
  updateEnterpriseRequestStatus,
  deleteEnterpriseRequest,
} from '@/app/lib/supabase/server';
import type { RequestStatus } from '@/app/lib/supabase/types';

export async function GET(request: NextRequest) {
  try {
    const { searchParams } = new URL(request.url);
    const status = searchParams.get('status') || undefined;
    const inquiry_type = searchParams.get('inquiry_type') || undefined;
    const limit = searchParams.get('limit') ? parseInt(searchParams.get('limit')!, 10) : 100;
    const offset = searchParams.get('offset') ? parseInt(searchParams.get('offset')!, 10) : 0;

    const result = await fetchEnterpriseRequests({
      status,
      inquiry_type,
      limit,
      offset,
    });

    return NextResponse.json({
      success: true,
      data: result.data,
      count: result.count,
      isMock: result.isMock ?? false,
    });
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : String(err);
    console.error('[ADMIN_REQUESTS_GET] Error:', msg);
    return NextResponse.json(
      { error: 'Failed to fetch enterprise requests' },
      { status: 500 }
    );
  }
}

export async function PATCH(request: NextRequest) {
  try {
    const body = await request.json().catch(() => null);
    if (!body || !body.id) {
      return NextResponse.json(
        { error: 'Request ID is required' },
        { status: 400 }
      );
    }

    const { id, status, admin_notes } = body;
    const validStatuses: RequestStatus[] = ['new', 'in_review', 'contacted', 'qualified', 'closed'];

    if (status && !validStatuses.includes(status)) {
      return NextResponse.json(
        { error: `Invalid status. Must be one of: ${validStatuses.join(', ')}` },
        { status: 400 }
      );
    }

    const result = await updateEnterpriseRequestStatus(
      id,
      status as RequestStatus,
      typeof admin_notes === 'string' ? admin_notes : undefined
    );

    if (!result.success) {
      return NextResponse.json(
        { error: result.error || 'Failed to update request' },
        { status: 400 }
      );
    }

    return NextResponse.json({
      success: true,
      data: result.data,
    });
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : String(err);
    console.error('[ADMIN_REQUESTS_PATCH] Error:', msg);
    return NextResponse.json(
      { error: 'Failed to update request' },
      { status: 500 }
    );
  }
}

export async function DELETE(request: NextRequest) {
  try {
    const { searchParams } = new URL(request.url);
    let id = searchParams.get('id');

    if (!id) {
      const body = await request.json().catch(() => null);
      id = body?.id;
    }

    if (!id) {
      return NextResponse.json(
        { error: 'Request ID is required' },
        { status: 400 }
      );
    }

    const result = await deleteEnterpriseRequest(id);

    if (!result.success) {
      return NextResponse.json(
        { error: result.error || 'Failed to delete request' },
        { status: 400 }
      );
    }

    return NextResponse.json({
      success: true,
      message: 'Request deleted successfully',
    });
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : String(err);
    console.error('[ADMIN_REQUESTS_DELETE] Error:', msg);
    return NextResponse.json(
      { error: 'Failed to delete request' },
      { status: 500 }
    );
  }
}
