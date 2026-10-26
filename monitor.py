import psutil

def get_system_metrics_text():
    """Gathers live system metrics and returns them as a single string."""
    data = ["--- Live System Metrics ---"]
    
    # 1. CPU
    cpu_usage = psutil.cpu_percent(interval=1)
    data.append(f"CPU Usage: {cpu_usage}%")
    
    # 2. RAM
    ram = psutil.virtual_memory()
    data.append(f"RAM Usage: {ram.percent}% ({ram.used / (1024**3):.2f} GB / {ram.total / (1024**3):.2f} GB)")
    
    # 3. Battery
    battery = psutil.sensors_battery()
    if battery:
        status = "Plugged In" if battery.power_plugged else "On Battery"
        data.append(f"Battery: {battery.percent:.1f}% ({status})")
        
    # 4. Temperature
    if hasattr(psutil, "sensors_temperatures"):
        temps = psutil.sensors_temperatures()
        if temps:
            for name, entries in temps.items():
                for entry in entries:
                    data.append(f"{name} Temp: {entry.current}°C")
                    break 
                    
    return "\n".join(data)

def get_top_processes_text(limit=5):
    """Gathers the most resource-hungry processes currently running."""
    data = [f"--- Top {limit} Resource-Heavy Processes ---"]
    processes = []
    
    # Iterate over all running processes safely
    for proc in psutil.process_iter(['pid', 'name', 'cpu_percent', 'memory_percent']):
        try:
            processes.append(proc.info)
        except (psutil.NoSuchProcess, psutil.AccessDenied, psutil.ZombieProcess):
            pass # Ignore processes that close while we are reading them
            
    # Sort by memory usage first (you can change this to cpu_percent if you prefer)
    processes = sorted(processes, key=lambda p: p['memory_percent'] or 0, reverse=True)
    
    # Format the top N processes
    for p in processes[:limit]:
        cpu = p['cpu_percent'] or 0.0
        mem = p['memory_percent'] or 0.0
        data.append(f"PID: {p['pid']} | Name: {p['name']} | CPU: {cpu:.1f}% | RAM: {mem:.1f}%")
        
    return "\n".join(data)

def find_process_by_name(search_term: str) -> str:
    """Searches for running processes that match a specific name."""
    data = [f"--- Process Search Results for '{search_term}' ---"]
    found = False
    
    for proc in psutil.process_iter(['pid', 'name', 'cmdline']):
        try:
            name = proc.info.get('name') or ""
            # Search the process name
            if search_term.lower() in name.lower():
                data.append(f"PID: {proc.info['pid']} | Name: {name}")
                found = True
        except (psutil.NoSuchProcess, psutil.AccessDenied, psutil.ZombieProcess):
            pass
            
    if not found:
        return f"❌ No processes found matching '{search_term}'."
    return "\n".join(data)

if __name__ == "__main__":
    # Test it
    print(get_system_metrics_text())
    print("\n")
    print(get_top_processes_text())
