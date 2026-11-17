import { describe, it, expect, vi } from "vitest";
import React from "react";
import { render, screen, fireEvent } from "@testing-library/react";
import { DesktopKeyboardCheatsheet } from "@/components/ui/DesktopKeyboardCheatsheet";

describe("Warehouse Portal DesktopKeyboardCheatsheet Component", () => {
  it("does not render when open is false", () => {
    const { container } = render(
      <DesktopKeyboardCheatsheet open={false} onClose={vi.fn()} />,
    );
    expect(container.firstChild).toBeNull();
  });

  it("renders warehouse keyboard shortcuts dialog when open is true", () => {
    render(
      <DesktopKeyboardCheatsheet open={true} onClose={vi.fn()} />,
    );
    expect(screen.getByText("Desktop Keyboard Shortcuts")).toBeDefined();
    expect(screen.getByText("Global Navigation")).toBeDefined();
    expect(screen.getByText("Open Command Palette / Search")).toBeDefined();
    expect(screen.getByText("Trigger Blind Recount Ratchet")).toBeDefined();
    expect(screen.getByText("Open 8-Point Pre-Trip DVIR Walkaround")).toBeDefined();
  });

  it("triggers onClose when clicking the close button or pressing Escape", () => {
    const onClose = vi.fn();
    render(<DesktopKeyboardCheatsheet open={true} onClose={onClose} />);
    
    const closeBtn = screen.getByLabelText("Close shortcuts");
    fireEvent.click(closeBtn);
    expect(onClose).toHaveBeenCalledTimes(1);

    fireEvent.keyDown(window, { key: "Escape" });
    expect(onClose).toHaveBeenCalledTimes(2);
  });
});
