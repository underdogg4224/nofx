import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { useSystemConfig } from './useSystemConfig';
import * as configLib from '../lib/config';

// Mock the config module
vi.mock('../lib/config', () => ({
  getSystemConfig: vi.fn(),
}));

describe('useSystemConfig', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('fetches and returns system config successfully', async () => {
    const mockConfig = {
      version: '1.0.0',
      apiEndpoint: 'https://api.example.com',
      features: {
        trading: true,
        analytics: true,
      },
    };

    vi.mocked(configLib.getSystemConfig).mockResolvedValue(mockConfig);

    const { result } = renderHook(() => useSystemConfig());

    // Initially loading
    expect(result.current.loading).toBe(true);
    expect(result.current.config).toBeNull();
    expect(result.current.error).toBeNull();

    // Wait for config to load
    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(result.current.config).toEqual(mockConfig);
    expect(result.current.error).toBeNull();
    expect(configLib.getSystemConfig).toHaveBeenCalledTimes(1);
  });

  it('handles fetch error correctly', async () => {
    const errorMessage = 'Failed to fetch config';
    vi.mocked(configLib.getSystemConfig).mockRejectedValue(new Error(errorMessage));

    const { result } = renderHook(() => useSystemConfig());

    // Initially loading
    expect(result.current.loading).toBe(true);

    // Wait for error state
    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(result.current.config).toBeNull();
    expect(result.current.error).toBe(errorMessage);
    expect(configLib.getSystemConfig).toHaveBeenCalledTimes(1);
  });

  it('cleans up on unmount', async () => {
    const mockConfig = {
      version: '1.0.0',
      apiEndpoint: 'https://api.example.com',
    };

    // Create a promise that we control
    let resolveConfig: (value: any) => void;
    const configPromise = new Promise((resolve) => {
      resolveConfig = resolve;
    });

    vi.mocked(configLib.getSystemConfig).mockReturnValue(configPromise as any);

    const { result, unmount } = renderHook(() => useSystemConfig());

    // Unmount before promise resolves
    unmount();

    // Resolve the promise after unmount
    resolveConfig!(mockConfig);

    // Wait a bit to ensure state doesn't update after unmount
    await new Promise((resolve) => setTimeout(resolve, 100));

    // The state should still be in loading state since component was unmounted
    expect(result.current.loading).toBe(true);
  });

  it('only fetches config once on mount', async () => {
    const mockConfig = {
      version: '1.0.0',
    };

    vi.mocked(configLib.getSystemConfig).mockResolvedValue(mockConfig);

    const { result, rerender } = renderHook(() => useSystemConfig());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    // Rerender the hook
    rerender();

    // Should still only have been called once
    expect(configLib.getSystemConfig).toHaveBeenCalledTimes(1);
  });
});
