import axios from 'axios';
import type { Device } from '../components/DeviceModal';

export const fetchDevices = () => axios.get<Device[]>('/api/devices');
export const createDevice = (data: Device) => axios.post('/api/devices', data);
export const updateDevice = (id: string, data: Device) => axios.put(`/api/devices/${id}`, data);
export const deleteDevice = (id: string) => axios.delete(`/api/devices/${id}`);
