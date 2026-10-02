import { jest, beforeEach, it, expect } from '@jest/globals';
import { downloadAndSharePdf } from './download-pdf';
import { useAuthStore } from '../store/auth-store';
import { api } from './api';
import { File } from 'expo-file-system';
import * as Sharing from 'expo-sharing';
import { showAlertOutsideReact } from '@homechef/mobile-shared/ui';
jest.mock('../store/auth-store', () => ({ useAuthStore: { getState: jest.fn() } }));
jest.mock('./api',()=>({api:{get:jest.fn()}}));
jest.mock('expo-file-system', () => ({Paths:{cache:'file:///cache/'},File:jest.fn().mockImplementation(()=>({uri:'file:///cache/tax.pdf',write:jest.fn()}))}));
jest.mock('expo-file-system/legacy', () => ({ downloadAsync: jest.fn<()=>Promise<unknown>>().mockRejectedValue(new Error('Native download unavailable')) }));
jest.mock('expo-sharing', () => ({ isAvailableAsync: jest.fn(), shareAsync: jest.fn() }));
jest.mock('@homechef/mobile-shared/ui', () => ({ showAlertOutsideReact: jest.fn() }));
beforeEach(() => {
 jest.clearAllMocks();
 (useAuthStore.getState as jest.Mock).mockReturnValue({accessToken:'test-session-token'});
 (api.get as jest.Mock<()=>Promise<unknown>>).mockResolvedValue({status:200,data:new Uint8Array([37,80,68,70]).buffer});
 (Sharing.isAvailableAsync as jest.Mock<()=>Promise<boolean>>).mockResolvedValue(true);
});
it('downloads through the authenticated API client and writes PDF bytes',async()=>{
 await downloadAndSharePdf('/chef/tax/fy-statement.pdf?year=2026','tax.pdf');
 expect(api.get).toHaveBeenCalledWith('/chef/tax/fy-statement.pdf?year=2026',{responseType:'arraybuffer'});
 expect(File).toHaveBeenCalledWith('file:///cache/','tax.pdf');
 expect(Sharing.shareAsync).toHaveBeenCalledWith('file:///cache/tax.pdf',expect.objectContaining({mimeType:'application/pdf'}));
});
it('requires an authenticated app session',async()=>{
 (useAuthStore.getState as jest.Mock).mockReturnValue({accessToken:null});
 await downloadAndSharePdf('/chef/tax/fy-statement.pdf','tax.pdf');
 expect(api.get).not.toHaveBeenCalled();
 expect(showAlertOutsideReact).toHaveBeenCalledWith('Sign in required',expect.any(String));
});
it('never shares an unsuccessful download',async()=>{
 (api.get as jest.Mock<()=>Promise<unknown>>).mockRejectedValue(new Error('Request failed with status code 401'));
 await downloadAndSharePdf('/chef/tax/fy-statement.pdf','tax.pdf');
 expect(Sharing.shareAsync).not.toHaveBeenCalled();
 expect(showAlertOutsideReact).toHaveBeenCalledWith('Could not download','Request failed with status code 401');
});
