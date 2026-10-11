import { expect } from 'chai';
import { shallowMount } from '@vue/test-utils';
import InventoryForm from '@/components/InventoryForm.vue';
import mockAxios from './helpers/axiosMock';

const settle = () => new Promise((resolve) => { setTimeout(resolve, 10); });

describe('Inventory file suggestions', () => {
  let http;
  let wrapper;
  let inventoryResponse;

  beforeEach(async () => {
    http = mockAxios();
    inventoryResponse = () => ['inventories/dev.ini', 'inventories/prod.yaml', 'hosts'];
    http.respond((request) => {
      if (request.url.endsWith('/inventory-files')) return inventoryResponse(request);
      return [];
    });
    wrapper = shallowMount(InventoryForm, {
      propsData: { projectId: 1, itemId: 'new', features: {} },
      mocks: { $t: (key) => key },
    });
    await settle();
    await wrapper.setData({
      item: {
        name: 'Inventory', type: 'file', inventory: 'hosts', repository_id: null,
      },
    });
  });

  afterEach(() => {
    wrapper.destroy();
    http.restore();
  });

  it('keeps the repository optional and supports a manually entered path', async () => {
    const input = wrapper.findAll('v-text-field-stub').wrappers
      .find((field) => field.attributes('label') === 'pathToInventoryFile');
    input.vm.$emit('input', 'inventories/custom.conf');
    await settle();
    expect(wrapper.vm.item.inventory).to.equal('inventories/custom.conf');
    expect(wrapper.find('v-combobox-stub').exists()).to.equal(false);
    expect(http.requests.filter((r) => r.url.endsWith('/inventory-files'))).to.have.length(0);
  });

  it('loads paths on repository selection and allows a path outside the suggestions', async () => {
    await wrapper.setData({ item: { ...wrapper.vm.item, repository_id: 2 } });
    await settle();
    expect(http.requests.some((r) => r.url === '/api/project/1/repositories/2/inventory-files')).to.equal(true);
    expect(wrapper.vm.inventoryFiles).to.deep.equal(['inventories/dev.ini', 'inventories/prod.yaml', 'hosts']);
    const picker = wrapper.find('v-combobox-stub');
    picker.vm.$emit('input', 'inventories/dev.ini');
    await settle();
    expect(wrapper.vm.item.inventory).to.equal('inventories/dev.ini');
    picker.vm.$emit('update:search-input', 'scripts/dynamic_inventory.py');
    picker.vm.$emit('update:search-input', null);
    await settle();
    expect(wrapper.vm.item.inventory).to.equal('scripts/dynamic_inventory.py');

    await wrapper.setData({ item: { ...wrapper.vm.item, repository_id: null } });
    await settle();
    expect(wrapper.vm.inventoryFiles).to.deep.equal([]);
    expect(wrapper.vm.item.inventory).to.equal('scripts/dynamic_inventory.py');
    expect(wrapper.find('v-combobox-stub').exists()).to.equal(false);
  });

  it('cancels stale requests when switching repositories and keeps the new results', async () => {
    const pending = [];
    inventoryResponse = (request) => new Promise((resolve) => {
      pending.push({ resolve, signal: request.signal });
    });
    await wrapper.setData({ item: { ...wrapper.vm.item, repository_id: 2 } });
    await settle();
    await wrapper.setData({ item: { ...wrapper.vm.item, repository_id: 3 } });
    await settle();
    expect(pending[0].signal.aborted).to.equal(true);
    pending[1].resolve(['new/hosts.ini']);
    await settle();
    pending[0].resolve(['old/hosts.ini']);
    await settle();
    expect(wrapper.vm.inventoryFiles).to.deep.equal(['new/hosts.ini']);
    expect(wrapper.vm.inventoryFilesLoading).to.equal(false);
    expect(wrapper.vm.inventoryFilesError).to.equal(false);
  });

  it('keeps manual entry available after a loading error and can retry', async () => {
    inventoryResponse = () => { throw new Error('Repository unavailable'); };
    await wrapper.setData({ item: { ...wrapper.vm.item, repository_id: 2 } });
    await settle();
    expect(wrapper.vm.inventoryFilesError).to.equal(true);
    expect(wrapper.find('v-combobox-stub').attributes('hint')).to.equal('inventoryFilesLoadError');
    expect(wrapper.vm.item.inventory).to.equal('hosts');
    inventoryResponse = () => ['hosts.ini'];
    await wrapper.vm.loadInventoryFiles();
    expect(wrapper.vm.inventoryFiles).to.deep.equal(['hosts.ini']);
    expect(wrapper.vm.inventoryFilesError).to.equal(false);
  });

  it('stops loading and drops suggestions when switching to a static inventory', async () => {
    let signal;
    let finish;
    inventoryResponse = (request) => new Promise((resolve) => {
      signal = request.signal;
      finish = resolve;
    });
    await wrapper.setData({ item: { ...wrapper.vm.item, repository_id: 2 } });
    await settle();
    await wrapper.setData({ item: { ...wrapper.vm.item, type: 'static' } });
    await settle();
    expect(signal.aborted).to.equal(true);
    finish(['hosts.ini']);
    await settle();
    expect(wrapper.vm.inventoryFiles).to.deep.equal([]);
    expect(wrapper.vm.inventoryFilesLoading).to.equal(false);
    expect(wrapper.vm.inventoryFilesError).to.equal(false);
  });
});
