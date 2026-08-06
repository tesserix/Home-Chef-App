import { useDeliveryStore } from '../delivery-store';

describe('useDeliveryStore', () => {
  beforeEach(() => {
    useDeliveryStore.setState({
      isTrackingLocation: false,
      activeDeliveryId: null,
    });
  });

  it('records the delivery being tracked', () => {
    useDeliveryStore.getState().setTrackingLocation(true, 'del-1');

    expect(useDeliveryStore.getState()).toMatchObject({
      isTrackingLocation: true,
      activeDeliveryId: 'del-1',
    });
  });

  it('clears the active delivery when tracking stops', () => {
    useDeliveryStore.getState().setTrackingLocation(true, 'del-1');
    useDeliveryStore.getState().setTrackingLocation(false);

    expect(useDeliveryStore.getState()).toMatchObject({
      isTrackingLocation: false,
      activeDeliveryId: null,
    });
  });

  // A caller that starts tracking without an id must not inherit the previous
  // delivery's id — that would attach location pings to the wrong delivery.
  it('does not carry over the previous id when tracking restarts without one', () => {
    useDeliveryStore.getState().setTrackingLocation(true, 'del-1');
    useDeliveryStore.getState().setTrackingLocation(false);
    useDeliveryStore.getState().setTrackingLocation(true);

    expect(useDeliveryStore.getState().activeDeliveryId).toBeNull();
  });
});
