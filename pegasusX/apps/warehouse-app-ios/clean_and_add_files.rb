require 'xcodeproj'
project_path = 'WarehouseAppIOS.xcodeproj'
project = Xcodeproj::Project.open(project_path)
target = project.targets.first

# Clean up broken references
['DemandForecastView.swift', 'ForecastConfidenceView.swift'].each do |file_name|
  bad_path = 'WarehouseApp/WarehouseApp/Views/Components/' + file_name
  # Find and remove any file references with broken path or just the same file name
  project.files.select { |f| f.path == bad_path || f.path == file_name || f.path == '../../WarehouseApp/Views/Components/' + file_name }.each do |ref|
    ref.build_files.each(&:remove_from_project)
    ref.remove_from_project
  end
end

group1 = project.main_group.find_subpath('WarehouseApp/Views/Components', true)
compile_phase = target.source_build_phase

file_ref1 = group1.new_file('ForecastConfidenceView.swift')
compile_phase.add_file_reference(file_ref1, true)

group2 = project.main_group.find_subpath('WarehouseApp/Views/DemandForecast', true)
file_ref2 = group2.new_file('DemandForecastView.swift')
compile_phase.add_file_reference(file_ref2, true)

project.save
