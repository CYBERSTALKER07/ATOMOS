import os
import re

ROOT = "/Users/shakhzod/Desktop/V.O.I.D/pegasusX"

def read_file(path):
    with open(path, "r") as f:
        return f.read()

def write_file(path, content):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w") as f:
        f.write(content)

# 1. Desktop
try:
    desktop_page_path = os.path.join(ROOT, "apps/retailer-app-desktop/app/(dashboard)/catalog/page.tsx")
    desktop_comp_dir = os.path.join(ROOT, "apps/retailer-app-desktop/components/catalog")
    desktop_content = read_file(desktop_page_path)

    filters_idx = desktop_content.find('<div className="mb-8 flex flex-col gap-6">')
    grid_start_idx = desktop_content.find('<AnimatePresence mode="popLayout">')
    grid_end_idx = desktop_content.find('</AnimatePresence>') + len('</AnimatePresence>')
    stockbadge_idx = desktop_content.find('function StockBadge')

    if filters_idx != -1:
        filters_end_idx = desktop_content.find('      <div className="flex gap-8">')
        filters_tsx = desktop_content[filters_idx:filters_end_idx].strip()
        
        write_file(os.path.join(desktop_comp_dir, "CatalogFilters.tsx"), f'''import {{ Search, SlidersHorizontal }} from "lucide-react";
import {{ Skeleton }} from "../../../components/Skeleton";
import type {{ Supplier }} from "../../../lib/types";

export interface CatalogFiltersProps {{
  searchQuery: string;
  setSearchQuery: (query: string) => void;
  hasActiveFilters: boolean;
  clearFilters: () => void;
  categoryTabs: string[];
  activeCategory: string;
  setActiveCategory: (category: string) => void;
  categorySuppliersLoading: boolean;
  categorySuppliersError: string | null;
  categorySuppliers: Supplier[];
  activeSupplier: string;
  setActiveSupplier: (supplierId: string) => void;
  supplierList: Supplier[];
}}

export function CatalogFilters({{
  searchQuery,
  setSearchQuery,
  hasActiveFilters,
  clearFilters,
  categoryTabs,
  activeCategory,
  setActiveCategory,
  categorySuppliersLoading,
  categorySuppliersError,
  categorySuppliers,
  activeSupplier,
  setActiveSupplier,
  supplierList
}}: CatalogFiltersProps) {{
  return (
    {filters_tsx}
  );
}}
''')

    if grid_start_idx != -1:
        grid_tsx = desktop_content[grid_start_idx:grid_end_idx].strip()
        stock_badge_code = desktop_content[stockbadge_idx:].strip() if stockbadge_idx != -1 else ""
        stock_badge_code = stock_badge_code.replace("function StockBadge", "export function StockBadge")
        
        write_file(os.path.join(desktop_comp_dir, "ProductGrid.tsx"), f'''import {{ motion, AnimatePresence }} from "framer-motion";
import {{ Package }} from "lucide-react";
import {{ Skeleton }} from "../../../components/Skeleton";
import EmptyState from "../../../components/EmptyState";
import {{ isCatalogBlocked }} from "../../../lib/stock-policy";
import {{ productDisplayPrice, productListPrice, productSalePrice }} from "../../../lib/types";
import type {{ Product }} from "../../../lib/types";

export interface EmptyStateConfig {{
  headline: string;
  body: string;
  variant: "restricted" | "offline" | "error" | "no-products" | "no-results";
}}

export interface ProductGridProps {{
  loadingProducts: boolean;
  filteredProducts: Product[];
  emptyStateConfig: EmptyStateConfig;
  loadIssue: "restricted" | "offline" | "error" | null;
  refreshAll: () => void;
  hasActiveFilters: boolean;
  clearFilters: () => void;
  setSelectedProduct: (p: Product) => void;
  addToCart: (p: Product) => void;
}}

export function ProductGrid({{
  loadingProducts,
  filteredProducts,
  emptyStateConfig,
  loadIssue,
  refreshAll,
  hasActiveFilters,
  clearFilters,
  setSelectedProduct,
  addToCart
}}: ProductGridProps) {{
  return (
    {grid_tsx}
  );
}}

{stock_badge_code}
''')

    # Update desktop_page_path
    new_desktop = desktop_content
    if stockbadge_idx != -1:
        new_desktop = new_desktop[:stockbadge_idx]

    new_desktop = new_desktop[:grid_start_idx] + '''<ProductGrid
            loadingProducts={loadingProducts}
            filteredProducts={filteredProducts}
            emptyStateConfig={emptyStateConfig}
            loadIssue={loadIssue}
            refreshAll={refreshAll}
            hasActiveFilters={hasActiveFilters}
            clearFilters={clearFilters}
            setSelectedProduct={setSelectedProduct}
            addToCart={addToCart}
          />
          ''' + new_desktop[grid_end_idx:]

    filters_end_idx = new_desktop.find('      <div className="flex gap-8">')
    new_desktop = new_desktop[:filters_idx] + '''<CatalogFilters
        searchQuery={searchQuery}
        setSearchQuery={setSearchQuery}
        hasActiveFilters={hasActiveFilters}
        clearFilters={clearFilters}
        categoryTabs={categoryTabs}
        activeCategory={activeCategory}
        setActiveCategory={setActiveCategory}
        categorySuppliersLoading={categorySuppliersLoading}
        categorySuppliersError={categorySuppliersError}
        categorySuppliers={categorySuppliers}
        activeSupplier={activeSupplier}
        setActiveSupplier={setActiveSupplier}
        supplierList={supplierList}
      />
''' + new_desktop[filters_end_idx:]

    new_desktop = new_desktop.replace('import { PageSection } from "../../../components/PageSection";', 
    '''import { CatalogFilters } from "../../../components/catalog/CatalogFilters";
import { ProductGrid } from "../../../components/catalog/ProductGrid";
import { PageSection } from "../../../components/PageSection";''')

    write_file(desktop_page_path, new_desktop)
    print("Desktop done")
except Exception as e:
    print("Error in Desktop:", e)


# 2. Android
try:
    android_screen_path = os.path.join(ROOT, "apps/retailer-app-android/app/src/main/java/com/pegasusx/retailer/ui/screens/procurement/ProcurementScreen.kt")
    android_comp_dir = os.path.join(ROOT, "apps/retailer-app-android/app/src/main/java/com/pegasusx/retailer/ui/screens/procurement/components")
    android_content = read_file(android_screen_path)

    def extract_kotlin_component(name):
        idx = android_content.find(f"fun {name}(")
        if idx == -1: return None, ""
        
        comp_idx = android_content.rfind("@Composable", 0, idx)
        if comp_idx == -1: comp_idx = idx - 12
        
        next_comp_idx = android_content.find("@Composable", idx)
        next_fun_idx = android_content.find("private fun", idx)
        
        end_idx = len(android_content)
        if next_comp_idx != -1 and next_comp_idx < end_idx: end_idx = next_comp_idx
        if next_fun_idx != -1 and next_fun_idx < end_idx: end_idx = next_fun_idx
        
        code = android_content[comp_idx:end_idx].strip()
        return code, android_content[comp_idx:end_idx]

    components = ["ProcurementHeader", "StatBlock", "SuggestionCard", "SelectedSummary", "ProcurementActionBar", "QuantityStepper"]
    for comp in components:
        code, original_chunk = extract_kotlin_component(comp)
        if code:
            code = code.replace("private fun", "fun")
            write_file(os.path.join(android_comp_dir, f"{comp}.kt"), f'''package com.pegasusx.retailer.ui.screens.procurement.components

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.*
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.pegasusx.retailer.data.model.DemandForecast
import com.pegasusx.retailer.ui.screens.procurement.ProcurementUiState
import com.pegasusx.retailer.ui.theme.SoftSquircleShape

{code}
''')

    new_android_content = android_content
    header_idx = new_android_content.find("@Composable\nprivate fun ProcurementHeader")
    if header_idx != -1:
        new_android_content = new_android_content[:header_idx].strip() + "\n"

    new_android_content = new_android_content.replace(
        "import com.pegasusx.retailer.ui.theme.SoftSquircleShape",
        "import com.pegasusx.retailer.ui.theme.SoftSquircleShape\nimport com.pegasusx.retailer.ui.screens.procurement.components.*"
    )

    write_file(android_screen_path, new_android_content)
    print("Android done")
except Exception as e:
    print("Error in Android:", e)


# 3. iOS
try:
    ios_view_path = os.path.join(ROOT, "apps/retailer-app-ios/retailerapp/reatilerapp/Screens/CatalogView.swift")
    ios_comp_dir = os.path.join(ROOT, "apps/retailer-app-ios/retailerapp/reatilerapp/Components/Catalog")
    ios_content = read_file(ios_view_path)

    enum_match = re.search(r'(private enum CatalogBrowseMode.*?^})\n', ios_content, re.DOTALL | re.MULTILINE)
    if enum_match:
        ios_content = ios_content.replace(enum_match.group(0), "")
        write_file(os.path.join(ios_comp_dir, "CatalogBrowseMode.swift"), "import Foundation\n\n" + enum_match.group(1).replace("private enum", "enum") + "\n")

    def extract_ios_view(name, struct_args, additional_funcs=[]):
        global ios_content
        pattern = rf'(    private var {name}: some View {{.*?^    }}\n)'
        match = re.search(pattern, ios_content, re.DOTALL | re.MULTILINE)
        if not match: return
        
        body = match.group(1).replace(f"private var {name}: some View {{", "var body: some View {")
        
        extra_funcs = ""
        for func_name in additional_funcs:
            f_pattern = rf'(    private func {func_name}.*?^    }}\n)'
            f_match = re.search(f_pattern, ios_content, re.DOTALL | re.MULTILINE)
            if f_match:
                extra_funcs += "\n\n" + f_match.group(1).replace("private func", "func")
                ios_content = ios_content.replace(f_match.group(0), "")
        
        code = f'''import SwiftUI

struct Catalog{name[0].upper() + name[1:]}: View {{
{struct_args}

{body.rstrip()}{extra_funcs}
}}
'''
        write_file(os.path.join(ios_comp_dir, f"Catalog{name[0].upper() + name[1:]}.swift"), code)
        ios_content = ios_content.replace(match.group(0), "")

    ios_content = ios_content.replace("            searchBar\n", "            CatalogSearchBar(searchText: $searchText, showFullSearch: $showFullSearch)\n")
    ios_content = ios_content.replace("                browseChips\n", "                CatalogBrowseChips(browseMode: $browseMode, onNavigateToSuppliers: onNavigateToSuppliers)\n")
    ios_content = ios_content.replace("                    allProductsGrid\n", "                    CatalogAllProductsGrid(products: products, selectedProduct: $selectedProduct)\n")
    ios_content = ios_content.replace("                    bentoGrid\n", "                    CatalogBentoGrid(categories: categories)\n")
    ios_content = ios_content.replace("                        searchResults\n", "                        CatalogSearchResults(filteredProducts: filteredProducts, selectedProduct: $selectedProduct)\n")
    ios_content = ios_content.replace("                        noResultsState\n", "                        CatalogNoResultsState(searchText: searchText)\n")

    extract_ios_view("searchBar", "    @Binding var searchText: String\n    @Binding var showFullSearch: Bool")
    extract_ios_view("browseChips", "    @Binding var browseMode: CatalogBrowseMode\n    var onNavigateToSuppliers: () -> Void", ["browseChip"])
    extract_ios_view("allProductsGrid", "    let products: [Product]\n    @Binding var selectedProduct: Product?")
    extract_ios_view("bentoGrid", "    let categories: [ProductCategory]", ["bentoBig", "bentoWide", "bentoSmall", "bentoCompact"])
    extract_ios_view("searchResults", "    let filteredProducts: [Product]\n    @Binding var selectedProduct: Product?")
    extract_ios_view("noResultsState", "    let searchText: String")

    ios_content = re.sub(r'    // MARK: - [^\n]+\n+', '', ios_content)
    write_file(ios_view_path, ios_content)
    print("iOS done")
except Exception as e:
    print("Error in iOS:", e)

