import { Check, PencilLine, X } from "lucide-react";
import { useState } from "react";
import type { Plan } from "../types";

interface PlanReviewProps {
  plan?: Plan;
  open: boolean;
  busy: boolean;
  onAccept: () => void;
  onEdit: (feedback: string) => void;
  onDismiss: () => void;
}

export function PlanReview({ plan, open, busy, onAccept, onEdit, onDismiss }: PlanReviewProps) {
  const [feedback, setFeedback] = useState("");

  if (!open) return null;

  return (
    <section className="plan-review" aria-label="审核研究方案">
      <div className="review-header">
        <div>
          <span className="eyebrow">方案确认</span>
          <h2>{plan?.title || "研究方案已生成"}</h2>
        </div>
        <button className="icon-button ghost" type="button" onClick={onDismiss} aria-label="收起方案审核">
          <X size={17} />
        </button>
      </div>

      {plan ? (
        <ol className="review-steps">
          {plan.steps.map((step, index) => (
            <li key={`${step.title}-${index}`}>
              <span>{index + 1}</span>
              <div>
                <b>{step.title}</b>
                <p>{step.description}</p>
              </div>
            </li>
          ))}
        </ol>
      ) : (
        <p className="muted">方案详情生成后会显示在这里。</p>
      )}

      <label className="feedback-field">
        <span>调整方向</span>
        <textarea
          value={feedback}
          onChange={(event) => setFeedback(event.target.value)}
          placeholder="例如：更关注居家训练、补充饮食建议、缩短到三条核心结论..."
          rows={3}
        />
      </label>

      <div className="review-actions">
        <button className="primary-button" type="button" onClick={onAccept} disabled={busy}>
          <Check size={16} />
          开始研究
        </button>
        <button className="secondary-button" type="button" onClick={() => onEdit(feedback)} disabled={busy || feedback.trim() === ""}>
          <PencilLine size={16} />
          按反馈重做
        </button>
      </div>
    </section>
  );
}
