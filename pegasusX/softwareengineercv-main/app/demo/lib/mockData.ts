export const MOCK_ORDERS = [
  { id: 'ORD-8910', item: 'Solar Panels', qty: 120, status: 'Picking', eta: '2026-07-04', origin: 'Supplier A', destination: 'Warehouse West' },
  { id: 'ORD-8911', item: 'Inverters', qty: 45, status: 'Packed', eta: '2026-07-04', origin: 'Supplier B', destination: 'Warehouse East' },
  { id: 'ORD-8912', item: 'Lithium Batteries', qty: 300, status: 'Shipped', eta: '2026-07-03', origin: 'Supplier C', destination: 'Retailer NY' },
  { id: 'ORD-8913', item: 'Copper Wire', qty: 50, status: 'Delivered', eta: '2026-07-01', origin: 'Supplier A', destination: 'Retailer LA' },
  { id: 'ORD-8914', item: 'Mounting Brackets', qty: 1000, status: 'Processing', eta: '2026-07-06', origin: 'Supplier D', destination: 'Warehouse West' },
];

export const MOCK_INVENTORY = [
  { sku: 'SP-400W', name: '400W Solar Panel', inStock: 450, reserved: 120, status: 'Healthy', location: 'Warehouse West' },
  { sku: 'INV-5K', name: '5kW Inverter', inStock: 12, reserved: 45, status: 'Low Stock', location: 'Warehouse East' },
  { sku: 'BAT-10K', name: '10kWh Battery', inStock: 85, reserved: 30, status: 'Healthy', location: 'Warehouse Central' },
  { sku: 'MNT-RF', name: 'Roof Mount Kit', inStock: 5, reserved: 20, status: 'Critical', location: 'Retailer NY' },
];

export const MOCK_GATES = [
  { gateId: 'G-01', status: 'Occupied', carrier: 'FreightX', scheduledTime: '10:00 AM', actualTime: '09:55 AM', type: 'Inbound' },
  { gateId: 'G-02', status: 'Available', carrier: '-', scheduledTime: '-', actualTime: '-', type: 'Outbound' },
  { gateId: 'G-03', status: 'Occupied', carrier: 'TransLog', scheduledTime: '11:30 AM', actualTime: '11:45 AM', type: 'Outbound' },
  { gateId: 'G-04', status: 'Maintenance', carrier: '-', scheduledTime: '-', actualTime: '-', type: 'Inbound' },
];

export const MOCK_DELIVERIES = [
  { routeId: 'RT-109', driver: 'Alice T.', status: 'On Route', progress: 75, stopsCompleted: 12, totalStops: 15 },
  { routeId: 'RT-110', driver: 'Bob M.', status: 'Delayed', progress: 30, stopsCompleted: 4, totalStops: 12 },
  { routeId: 'RT-111', driver: 'Charlie K.', status: 'On Route', progress: 90, stopsCompleted: 8, totalStops: 9 },
];
