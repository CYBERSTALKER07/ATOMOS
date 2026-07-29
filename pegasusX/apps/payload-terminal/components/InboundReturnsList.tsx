import React from 'react';
import { Text, Pressable, ScrollView } from 'react-native';
import type { InboundRow } from '../inboundReturns';

type Props = {
  theme: any;
  list: InboundRow[];
  selectable: boolean;
  selected: Set<string>;
  onToggle: (id: string) => void;
  emptyText: string;
};

export function InboundReturnsList({ theme: T, list, selectable, selected, onToggle, emptyText }: Props) {
  const renderRow = (row: InboundRow, selectable: boolean) => (
    <Pressable
      key={row.return_id}
      onPress={() => selectable && onToggle(row.return_id)}
      style={{
        borderWidth: 1,
        borderColor: selectable && selected.has(row.return_id) ? T.colors.tint : T.colors.separator,
        borderRadius: 12,
        padding: 14,
        backgroundColor: selectable && selected.has(row.return_id) ? `${T.colors.tint}11` : T.colors.secondaryBackground,
      }}
    >
      <Text style={{ fontWeight: '700', color: T.colors.label }}>{row.product_name}</Text>
      <Text style={{ fontSize: 12, color: T.colors.secondaryLabel, marginTop: 4 }}>
        {row.driver_name || 'Driver'} · {row.reason} · {row.received_qty}/{row.expected_qty}
      </Text>
      {row.barcode ? (
        <Text style={{ fontSize: 11, fontFamily: T.typography?.mono?.fontFamily, color: T.colors.secondaryLabel, marginTop: 4 }}>
          EAN {row.barcode}
        </Text>
      ) : null}
      <Text style={{ fontSize: 11, fontFamily: T.typography?.mono?.fontFamily, color: T.colors.tertiaryLabel, marginTop: 4 }}>
        {row.return_id.slice(0, 8)} · suggest {row.suggested_disposition}
      </Text>
    </Pressable>
  );

  return (
    <ScrollView contentContainerStyle={{ padding: 16, gap: 12 }}>
      {list.length === 0 ? (
        <Text style={{ color: T.colors.secondaryLabel, textAlign: 'center', marginTop: 40 }}>
          {emptyText}
        </Text>
      ) : (
        list.map(row => renderRow(row, selectable))
      )}
    </ScrollView>
  );
}
