import { describe, it, expect, jest, beforeEach } from '@jest/globals';

// #1038: expo-sharing was imported at module scope, so on a build whose native
// side predates the dependency the *import* threw — "Cannot find native module
// 'ExpoSharing'" — and took down the whole receipt screen before it rendered.
// Nothing here may touch the native modules until the share button is pressed.

// eslint-disable-next-line @typescript-eslint/no-explicit-any -- jest mock signatures
type AnyMock = (...args: any[]) => any;
const mockDownloadAsync = jest.fn<AnyMock>();
const mockIsAvailableAsync = jest.fn<AnyMock>();
const mockShareAsync = jest.fn<AnyMock>();

jest.mock('expo-file-system/legacy', () => ({
  get cacheDirectory() {
    return 'file:///cache/';
  },
  downloadAsync: (...args: unknown[]) => mockDownloadAsync(...args),
}));

jest.mock('expo-sharing', () => ({
  isAvailableAsync: () => mockIsAvailableAsync(),
  shareAsync: (...args: unknown[]) => mockShareAsync(...args),
}));

import { shareReceiptPdf, SHARING_UNAVAILABLE } from './share-pdf';

beforeEach(() => {
  jest.clearAllMocks();
  mockDownloadAsync.mockResolvedValue({ status: 200, uri: 'file:///cache/receipt.pdf' });
  mockIsAvailableAsync.mockResolvedValue(true);
  mockShareAsync.mockResolvedValue(undefined);
});

describe('shareReceiptPdf', () => {
  it('touches no native module until it is called — the crash guard', () => {
    expect(mockDownloadAsync).not.toHaveBeenCalled();
    expect(mockIsAvailableAsync).not.toHaveBeenCalled();
  });

  it('downloads the PDF into the cache and opens the share sheet', async () => {
    await shareReceiptPdf('https://cdn/invoice.pdf', 'Fe3dr-tax-invoice-HC1.pdf');

    expect(mockDownloadAsync).toHaveBeenCalledWith(
      'https://cdn/invoice.pdf',
      'file:///cache/Fe3dr-tax-invoice-HC1.pdf',
    );
    expect(mockShareAsync).toHaveBeenCalledWith(
      'file:///cache/receipt.pdf',
      expect.objectContaining({ mimeType: 'application/pdf' }),
    );
  });

  it('reports a failed download rather than sharing a broken file', async () => {
    mockDownloadAsync.mockResolvedValue({ status: 500, uri: '' });

    await expect(shareReceiptPdf('https://cdn/x.pdf', 'x.pdf')).rejects.toThrow('500');
    expect(mockShareAsync).not.toHaveBeenCalled();
  });

  // A build without the native module must degrade to "open it in the browser",
  // not crash — that is the difference between #1038 and a working screen.
  it('flags a missing native module instead of throwing it at the screen', async () => {
    mockIsAvailableAsync.mockRejectedValue(new Error("Cannot find native module 'ExpoSharing'"));

    await expect(shareReceiptPdf('https://cdn/x.pdf', 'x.pdf')).rejects.toMatchObject({
      code: SHARING_UNAVAILABLE,
    });
  });

  it('flags a device that simply cannot share', async () => {
    mockIsAvailableAsync.mockResolvedValue(false);

    await expect(shareReceiptPdf('https://cdn/x.pdf', 'x.pdf')).rejects.toMatchObject({
      code: SHARING_UNAVAILABLE,
    });
  });
});

// The file-system module is native too — its absence used to escape the
// fallback and surface as "Couldn't share the PDF" (#1038, second pass).
describe('shareReceiptPdf without a file-system module', () => {
  it('degrades to the browser rather than failing the share', async () => {
    mockDownloadAsync.mockRejectedValue(new Error("Cannot find native module 'ExpoFileSystem'"));

    await expect(shareReceiptPdf('https://cdn/x.pdf', 'x.pdf')).rejects.toMatchObject({
      code: SHARING_UNAVAILABLE,
    });
  });
});
