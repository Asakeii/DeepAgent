import { ImagePlus, Send, X } from "lucide-react";
import { ChangeEvent, ClipboardEvent, KeyboardEvent, useRef } from "react";
import { fileToDataURL } from "../lib/format";

interface ComposerProps {
  value: string;
  image?: string;
  busy: boolean;
  onChange: (value: string) => void;
  onImageChange: (value?: string) => void;
  onSend: () => void;
}

export function Composer({ value, image, busy, onChange, onImageChange, onSend }: ComposerProps) {
  const fileRef = useRef<HTMLInputElement | null>(null);

  async function handleFile(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    if (!file) return;
    onImageChange(await fileToDataURL(file));
    event.target.value = "";
  }

  async function handlePaste(event: ClipboardEvent<HTMLTextAreaElement>) {
    const item = [...event.clipboardData.items].find((entry) => entry.type.startsWith("image/"));
    const file = item?.getAsFile();
    if (!file) return;
    event.preventDefault();
    onImageChange(await fileToDataURL(file));
  }

  function handleKeyDown(event: KeyboardEvent<HTMLTextAreaElement>) {
    if (event.key === "Enter" && !event.shiftKey) {
      event.preventDefault();
      onSend();
    }
  }

  return (
    <footer className="composer-wrap">
      {image ? (
        <div className="image-preview">
          <img src={image} alt="待发送图片" />
          <button className="icon-button" type="button" onClick={() => onImageChange(undefined)} aria-label="移除图片">
            <X size={15} />
          </button>
        </div>
      ) : null}

      <div className="composer">
        <button className="icon-button" type="button" onClick={() => fileRef.current?.click()} aria-label="上传图片" disabled={busy}>
          <ImagePlus size={19} />
        </button>
        <input ref={fileRef} className="visually-hidden" type="file" accept="image/*" onChange={handleFile} />
        <textarea
          value={value}
          onChange={(event) => onChange(event.target.value)}
          onPaste={handlePaste}
          onKeyDown={handleKeyDown}
          placeholder="记录今天的点滴，或提出你想了解的问题..."
          rows={1}
          disabled={busy}
        />
        <button className="send-button" type="button" onClick={onSend} disabled={busy || (value.trim() === "" && !image)} aria-label="发送">
          <Send size={18} />
        </button>
      </div>
      <div className="composer-hint">Enter 发送 · Shift+Enter 换行 · 可粘贴或上传食物图片</div>
    </footer>
  );
}
