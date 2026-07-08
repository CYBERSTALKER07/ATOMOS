require 'xcodeproj'
project_path = 'WarehouseAppIOS.xcodeproj'
project = Xcodeproj::Project.open(project_path)
target = project.targets.first

# Find the Swift file compile phase
compile_phase = target.source_build_phase

# Add both files
['DemandForecastView.swift', 'ForecastConfidenceView.swift'].each do |file_name|
  file_ref = group.new_file(file_name)
  compile_phase.add_file_reference(file_ref, true)
end

project.save
