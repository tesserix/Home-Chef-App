// Sharing a receipt shares the PDF the API issues — never a re-typed text copy,
// which could disagree with the tax document it claims to be.

/** Cache filename for the shared document. */
export function receiptFileName(orderNumber: string, isTaxInvoice: boolean): string {
  const kind = isTaxInvoice ? 'tax-invoice' : 'receipt';
  const slug = orderNumber
    .replace(/[^A-Za-z0-9-]+/g, '-')
    .replace(/-+/g, '-')
    .replace(/^-|-$/g, '');
  return slug ? `Fe3dr-${kind}-${slug}.pdf` : `Fe3dr-${kind}.pdf`;
}
